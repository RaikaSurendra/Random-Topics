# Chapter 02: Bloom Filters - Low Level Design

## Class Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      BloomFilter<T>                             │
│                        <<interface>>                            │
├─────────────────────────────────────────────────────────────────┤
│ + mightContain(item: T): boolean                                │
│ + add(item: T): void                                            │
│ + addAll(items: Collection<T>): void                            │
│ + clear(): void                                                 │
│ + approximateElementCount(): long                               │
│ + expectedFalsePositiveRate(): double                           │
└─────────────────────────────────────────────────────────────────┘
                    △                         △
                    │                         │
                    │                         │
┌───────────────────┴───────────┐  ┌─────────┴─────────────────────┐
│     InMemoryBloomFilter       │  │      RedisBloomFilter         │
├───────────────────────────────┤  ├───────────────────────────────┤
│ - filter: BloomFilter<String> │  │ - redisTemplate: RedisTemplate│
│ - lock: ReentrantReadWriteLock│  │ - filterKey: String           │
│ - expectedInsertions: int     │  │ - expectedInsertions: int     │
│ - falsePositiveRate: double   │  │ - falsePositiveRate: double   │
├───────────────────────────────┤  ├───────────────────────────────┤
│ + mightContain(sku): boolean  │  │ + mightContain(sku): boolean  │
│ + add(sku): void              │  │ + add(sku): void              │
│ + rebuild(skus): void         │  │ + rebuild(skus): void         │
│ - createFilter(): BloomFilter │  │ - executeBloomCommand(): Object│
└───────────────────────────────┘  └───────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                   BloomProtectedService                         │
├─────────────────────────────────────────────────────────────────┤
│ - bloomFilter: BloomFilter<String>                              │
│ - cachedProductService: CachedProductService                    │
│ - productRepository: ProductRepository                          │
│ - metrics: BloomMetrics                                         │
├─────────────────────────────────────────────────────────────────┤
│ + getProduct(sku: String): ProductDTO                           │
│ + saveProduct(dto: ProductDTO): ProductDTO                      │
│ + deleteProduct(sku: String): void                              │
│ + rebuildFilter(): void                                         │
│ + getStats(): BloomStats                                        │
│ - checkBloomFilter(sku: String): boolean                        │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    PenetrationAttackDemo                        │
├─────────────────────────────────────────────────────────────────┤
│ - bloomProtectedService: BloomProtectedService                  │
│ - unprotectedService: ProductService                            │
│ - loadGenerator: LoadGenerator                                  │
├─────────────────────────────────────────────────────────────────┤
│ + runAttack(requests: int, protected: boolean): AttackResult    │
│ + measureFalsePositives(tests: int): FalsePositiveResult        │
│ - generateFakeSkus(count: int): List<String>                    │
└─────────────────────────────────────────────────────────────────┘
```

## Method Specifications

### InMemoryBloomFilter

#### Constructor

```java
public InMemoryBloomFilter(int expectedInsertions, double falsePositiveRate) {
    this.expectedInsertions = expectedInsertions;
    this.falsePositiveRate = falsePositiveRate;
    this.filter = BloomFilter.create(
        Funnels.stringFunnel(StandardCharsets.UTF_8),
        expectedInsertions,
        falsePositiveRate
    );
    this.lock = new ReentrantReadWriteLock();
}
```

#### mightContain

```java
public boolean mightContain(String sku) {
    lock.readLock().lock();
    try {
        return filter.mightContain(sku);
    } finally {
        lock.readLock().unlock();
    }
}
```

#### rebuild

```java
public void rebuild(Collection<String> skus) {
    BloomFilter<String> newFilter = createFilter();
    skus.forEach(newFilter::put);

    lock.writeLock().lock();
    try {
        this.filter = newFilter;
    } finally {
        lock.writeLock().unlock();
    }
}
```

### RedisBloomFilter

#### mightContain

```java
public boolean mightContain(String sku) {
    try {
        Object result = redisTemplate.execute((RedisCallback<Object>) connection -> {
            return connection.execute("BF.EXISTS",
                filterKey.getBytes(),
                sku.getBytes());
        });
        return result != null && ((Long) result) == 1;
    } catch (Exception e) {
        log.warn("RedisBloom check failed, allowing request: {}", e.getMessage());
        return true; // Fail open - allow request if Bloom unavailable
    }
}
```

#### rebuild

```java
public void rebuild(Collection<String> skus) {
    String tempKey = filterKey + ":temp:" + System.currentTimeMillis();

    try {
        // Create new filter
        redisTemplate.execute((RedisCallback<Object>) connection -> {
            connection.execute("BF.RESERVE",
                tempKey.getBytes(),
                String.valueOf(falsePositiveRate).getBytes(),
                String.valueOf(expectedInsertions).getBytes());
            return null;
        });

        // Add all SKUs in batches
        List<List<String>> batches = Lists.partition(new ArrayList<>(skus), 1000);
        for (List<String> batch : batches) {
            redisTemplate.execute((RedisCallback<Object>) connection -> {
                byte[][] args = new byte[batch.size() + 1][];
                args[0] = tempKey.getBytes();
                for (int i = 0; i < batch.size(); i++) {
                    args[i + 1] = batch.get(i).getBytes();
                }
                connection.execute("BF.MADD", args);
                return null;
            });
        }

        // Atomic swap
        redisTemplate.rename(tempKey, filterKey);

    } catch (Exception e) {
        redisTemplate.delete(tempKey);
        throw new RuntimeException("Failed to rebuild Bloom filter", e);
    }
}
```

### BloomProtectedService

#### getProduct

```java
public ProductDTO getProduct(String sku) {
    metrics.recordCheck();
    long start = System.nanoTime();

    // Step 1: Bloom filter check
    if (!checkBloomFilter(sku)) {
        metrics.recordRejection();
        log.debug("Bloom filter rejected SKU: {}", sku);
        throw new ProductNotFoundException(sku);
    }

    // Step 2: Standard cache lookup
    try {
        ProductDTO product = cachedProductService.getProduct(sku);
        metrics.recordLatency(System.nanoTime() - start);
        return product;
    } catch (ProductNotFoundException e) {
        // This was a false positive - the key passed Bloom but doesn't exist
        metrics.recordFalsePositive();
        throw e;
    }
}
```

#### saveProduct

```java
@Transactional
public ProductDTO saveProduct(ProductDTO dto) {
    // Save to database
    ProductDTO saved = productService.saveProduct(dto);

    // Add to Bloom filter
    bloomFilter.add(saved.getSku());

    // Update cache
    cachedProductService.updateProduct(saved);

    return saved;
}
```

## Sequence Diagrams

### Request Blocked by Bloom Filter

```
Client          Controller       BloomService       BloomFilter
  │                 │                 │                 │
  │ GET /products/FAKE-SKU            │                 │
  │────────────────▶│                 │                 │
  │                 │ getProduct()    │                 │
  │                 │────────────────▶│                 │
  │                 │                 │ mightContain()  │
  │                 │                 │────────────────▶│
  │                 │                 │    false        │
  │                 │                 │◀────────────────│
  │                 │  NotFoundException                │
  │                 │◀────────────────│                 │
  │    404 Not Found                  │                 │
  │◀────────────────│                 │                 │
```

### Request Allowed (Valid Key)

```
Client          Controller       BloomService       BloomFilter      Cache
  │                 │                 │                 │              │
  │ GET /products/SKU-001             │                 │              │
  │────────────────▶│                 │                 │              │
  │                 │ getProduct()    │                 │              │
  │                 │────────────────▶│                 │              │
  │                 │                 │ mightContain()  │              │
  │                 │                 │────────────────▶│              │
  │                 │                 │    true         │              │
  │                 │                 │◀────────────────│              │
  │                 │                 │ cachedService.get()            │
  │                 │                 │────────────────────────────────▶
  │                 │                 │         ProductDTO             │
  │                 │                 │◀────────────────────────────────
  │                 │   ProductDTO    │                 │              │
  │                 │◀────────────────│                 │              │
  │    200 OK + data                  │                 │              │
  │◀────────────────│                 │                 │              │
```

### Filter Rebuild

```
Scheduler       BloomService       Repository        BloomFilter
  │                 │                  │                 │
  │ rebuildFilter() │                  │                 │
  │────────────────▶│                  │                 │
  │                 │ findAllSkus()    │                 │
  │                 │─────────────────▶│                 │
  │                 │   List<String>   │                 │
  │                 │◀─────────────────│                 │
  │                 │ rebuild(skus)                      │
  │                 │───────────────────────────────────▶│
  │                 │                  │     OK          │
  │                 │◀───────────────────────────────────│
  │     OK          │                  │                 │
  │◀────────────────│                  │                 │
```

## Data Structures

### BloomStats

```java
@Data
@Builder
public class BloomStats {
    private long totalChecks;
    private long rejections;
    private long falsePositives;
    private double rejectionRate;
    private double falsePositiveRate;
    private long approximateSize;
    private double configuredFalsePositiveRate;

    public String getSummary() {
        return String.format(
            "Checks: %d, Rejections: %d (%.2f%%), False Positives: %d (%.4f%%)",
            totalChecks, rejections, rejectionRate * 100,
            falsePositives, falsePositiveRate * 100
        );
    }
}
```

### AttackResult

```java
@Data
@Builder
public class AttackResult {
    private int totalRequests;
    private boolean protected;
    private int blockedByBloom;
    private int reachedCache;
    private int reachedDatabase;
    private double avgLatencyMs;
    private double p99LatencyMs;
    private Duration totalDuration;
}
```

## Redis Commands Reference

| Operation | Command | Example |
|-----------|---------|---------|
| Create filter | BF.RESERVE | `BF.RESERVE myfilter 0.01 10000` |
| Add item | BF.ADD | `BF.ADD myfilter item1` |
| Add multiple | BF.MADD | `BF.MADD myfilter item1 item2` |
| Check item | BF.EXISTS | `BF.EXISTS myfilter item1` |
| Check multiple | BF.MEXISTS | `BF.MEXISTS myfilter item1 item2` |
| Get info | BF.INFO | `BF.INFO myfilter` |

## Error Handling

### BloomFilterUnavailableException

```java
// When Bloom filter cannot be accessed
// Default behavior: Fail open (allow request)
public boolean mightContain(String sku) {
    try {
        return doCheck(sku);
    } catch (Exception e) {
        log.warn("Bloom filter unavailable, failing open: {}", e.getMessage());
        return true; // Allow request to proceed
    }
}
```

### FilterRebuildException

```java
// When rebuild fails
// Recovery: Keep old filter, retry later
@Scheduled(fixedDelay = 300000) // 5 minutes
public void attemptRebuild() {
    try {
        rebuildFilter();
    } catch (FilterRebuildException e) {
        log.error("Filter rebuild failed, will retry: {}", e.getMessage());
        // Alert monitoring
    }
}
```

## Performance Considerations

### Guava BloomFilter

- **Memory**: O(m) where m = bits
- **Add**: O(k) where k = hash functions
- **Check**: O(k)
- **Thread safety**: Requires external synchronization

### RedisBloom

- **Memory**: O(m) in Redis
- **Network**: 1 round-trip per operation
- **Batching**: Use MADD/MEXISTS for bulk operations

### Optimization Tips

```java
// Batch additions for new products
public void addProducts(List<Product> products) {
    List<String> skus = products.stream()
        .map(Product::getSku)
        .toList();
    bloomFilter.addAll(skus);  // Single Redis call with MADD
}

// Pre-check before expensive operations
public List<ProductDTO> getProducts(List<String> skus) {
    List<String> possiblyExist = skus.stream()
        .filter(bloomFilter::mightContain)
        .toList();
    // Only query cache/DB for filtered list
    return cachedService.getProducts(possiblyExist);
}
```

## Testing Strategy

### Unit Tests

```java
@Test
void mightContain_nonExistentKey_returnsFalse() {
    InMemoryBloomFilter filter = new InMemoryBloomFilter(1000, 0.01);
    filter.add("existing-key");

    assertThat(filter.mightContain("non-existent")).isFalse();
}

@Test
void mightContain_existingKey_returnsTrue() {
    InMemoryBloomFilter filter = new InMemoryBloomFilter(1000, 0.01);
    filter.add("existing-key");

    assertThat(filter.mightContain("existing-key")).isTrue();
}
```

### Integration Tests

```java
@Test
void falsePositiveRate_withinBounds() {
    int total = 100_000;
    int falsePositives = 0;

    for (int i = 0; i < total; i++) {
        String fakeKey = "fake-" + UUID.randomUUID();
        if (bloomFilter.mightContain(fakeKey)) {
            falsePositives++;
        }
    }

    double actualRate = (double) falsePositives / total;
    assertThat(actualRate).isLessThan(0.02); // 2% threshold
}
```
