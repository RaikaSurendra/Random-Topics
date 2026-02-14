# Chapter 01: Caching Fundamentals - Low Level Design

## Class Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     ProductController                            │
├─────────────────────────────────────────────────────────────────┤
│ - cachedProductService: CachedProductService                    │
│ - productService: ProductService                                │
├─────────────────────────────────────────────────────────────────┤
│ + getProduct(sku: String): ResponseEntity<ProductDTO>           │
│ + getProductUncached(sku: String): ResponseEntity<ProductDTO>   │
│ + updateProduct(sku: String, dto: ProductDTO): ResponseEntity   │
│ + deleteProduct(sku: String): ResponseEntity<Void>              │
│ + evictCache(sku: String): ResponseEntity<Void>                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ uses
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   CachedProductService                          │
├─────────────────────────────────────────────────────────────────┤
│ - productRepository: ProductRepository                          │
│ - cacheMetrics: CacheMetrics                                    │
│ - redisTemplate: RedisTemplate<String, ProductDTO>              │
├─────────────────────────────────────────────────────────────────┤
│ + @Cacheable getProduct(sku: String): ProductDTO                │
│ + @CachePut updateProduct(dto: ProductDTO): ProductDTO          │
│ + @CacheEvict evictProduct(sku: String): void                   │
│ + @CacheEvict(allEntries) clearCache(): void                    │
│ - loadFromDatabase(sku: String): ProductDTO                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ extends
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      ProductService                             │
├─────────────────────────────────────────────────────────────────┤
│ - productRepository: ProductRepository                          │
├─────────────────────────────────────────────────────────────────┤
│ + getProduct(sku: String): ProductDTO                           │
│ + getAllProducts(): List<ProductDTO>                            │
│ + saveProduct(dto: ProductDTO): ProductDTO                      │
│ + deleteProduct(sku: String): void                              │
│ + existsBySku(sku: String): boolean                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ uses
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    ProductRepository                            │
├─────────────────────────────────────────────────────────────────┤
│ <<interface>> extends JpaRepository<Product, Long>              │
├─────────────────────────────────────────────────────────────────┤
│ + findBySku(sku: String): Optional<Product>                     │
│ + existsBySku(sku: String): boolean                             │
│ + findAllSkus(): List<String>                                   │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                     MetricsController                           │
├─────────────────────────────────────────────────────────────────┤
│ - cacheMetrics: CacheMetrics                                    │
├─────────────────────────────────────────────────────────────────┤
│ + getCacheStats(): ResponseEntity<CacheStats>                   │
│ + getAllCacheStats(): ResponseEntity<List<CacheStats>>          │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                       RedisConfig                               │
├─────────────────────────────────────────────────────────────────┤
│ - ttlSeconds: long                                              │
├─────────────────────────────────────────────────────────────────┤
│ + redisConnectionFactory(): LettuceConnectionFactory            │
│ + redisTemplate(): RedisTemplate<String, Object>                │
│ + cacheManager(): RedisCacheManager                             │
│ + cacheConfiguration(): RedisCacheConfiguration                 │
└─────────────────────────────────────────────────────────────────┘
```

## Method Specifications

### CachedProductService

#### getProduct

```java
@Cacheable(value = "products", key = "#sku")
public ProductDTO getProduct(String sku) {
    // Metrics are handled by custom cache listener
    return loadFromDatabase(sku);
}
```

**Behavior**:
1. Spring AOP intercepts the call
2. Generates cache key: `products::SKU-001`
3. Checks Redis for existing entry
4. If found (HIT): return cached value, skip method
5. If not found (MISS): execute method, cache result

**Cache Key Format**: `{cacheName}::{sku}`

#### updateProduct

```java
@CachePut(value = "products", key = "#dto.sku")
public ProductDTO updateProduct(ProductDTO dto) {
    Product product = productRepository.findBySku(dto.getSku())
        .orElseThrow(() -> new ProductNotFoundException(dto.getSku()));
    // Update fields
    Product saved = productRepository.save(product);
    return ProductDTO.from(saved);
}
```

**Behavior**:
1. Always executes the method
2. Stores return value in cache
3. Use for updates when you want the new value cached

#### evictProduct

```java
@CacheEvict(value = "products", key = "#sku")
public void evictProduct(String sku) {
    // Method body can be empty - eviction happens via AOP
    log.info("Evicted cache for SKU: {}", sku);
}
```

**Behavior**:
1. Removes entry from cache before/after method execution
2. `beforeInvocation = true` evicts before method runs
3. Default (false) evicts after method completes

### RedisConfig

#### cacheManager

```java
@Bean
public RedisCacheManager cacheManager(RedisConnectionFactory connectionFactory) {
    RedisCacheConfiguration config = RedisCacheConfiguration.defaultCacheConfig()
        .entryTtl(Duration.ofSeconds(ttlSeconds))
        .serializeKeysWith(RedisSerializationContext.SerializationPair
            .fromSerializer(new StringRedisSerializer()))
        .serializeValuesWith(RedisSerializationContext.SerializationPair
            .fromSerializer(new GenericJackson2JsonRedisSerializer()));

    return RedisCacheManager.builder(connectionFactory)
        .cacheDefaults(config)
        .withCacheConfiguration("products", config)
        .build();
}
```

**Configuration Options**:

| Option | Value | Purpose |
|--------|-------|---------|
| entryTtl | 60s | Auto-expire after 60 seconds |
| keySerializer | String | Human-readable keys |
| valueSerializer | JSON | Human-readable values |

## Sequence Diagrams

### Read with Cache Hit

```
Client          Controller       CachedService      Redis        DB
  │                 │                 │               │           │
  │ GET /products/001               │               │           │
  │────────────────▶│                 │               │           │
  │                 │ getProduct(001) │               │           │
  │                 │────────────────▶│               │           │
  │                 │                 │ GET products::001         │
  │                 │                 │──────────────▶│           │
  │                 │                 │    ProductDTO │           │
  │                 │                 │◀──────────────│           │
  │                 │   ProductDTO    │               │           │
  │                 │◀────────────────│               │           │
  │    200 OK       │                 │               │           │
  │◀────────────────│                 │               │           │
```

### Read with Cache Miss

```
Client          Controller       CachedService      Redis        DB
  │                 │                 │               │           │
  │ GET /products/001               │               │           │
  │────────────────▶│                 │               │           │
  │                 │ getProduct(001) │               │           │
  │                 │────────────────▶│               │           │
  │                 │                 │ GET products::001         │
  │                 │                 │──────────────▶│           │
  │                 │                 │     null      │           │
  │                 │                 │◀──────────────│           │
  │                 │                 │ SELECT * FROM products    │
  │                 │                 │──────────────────────────▶│
  │                 │                 │         Product           │
  │                 │                 │◀──────────────────────────│
  │                 │                 │ SET products::001         │
  │                 │                 │──────────────▶│           │
  │                 │   ProductDTO    │               │           │
  │                 │◀────────────────│               │           │
  │    200 OK       │                 │               │           │
  │◀────────────────│                 │               │           │
```

### Update with Cache Eviction

```
Client          Controller       CachedService      Redis        DB
  │                 │                 │               │           │
  │ PUT /products/001               │               │           │
  │────────────────▶│                 │               │           │
  │                 │ updateProduct() │               │           │
  │                 │────────────────▶│               │           │
  │                 │                 │ UPDATE products           │
  │                 │                 │──────────────────────────▶│
  │                 │                 │         OK                │
  │                 │                 │◀──────────────────────────│
  │                 │                 │ DEL products::001         │
  │                 │                 │──────────────▶│           │
  │                 │   ProductDTO    │               │           │
  │                 │◀────────────────│               │           │
  │    200 OK       │                 │               │           │
  │◀────────────────│                 │               │           │
```

## Redis Data Structure

### Key Format

```
products::{sku}
```

Examples:
- `products::SKU-001`
- `products::SKU-002`

### Value Format (JSON)

```json
{
  "@class": "com.learning.cache.common.dto.ProductDTO",
  "id": 1,
  "sku": "SKU-001",
  "name": "Wireless Mouse",
  "description": "Ergonomic wireless mouse",
  "price": 29.99,
  "category": "Electronics",
  "stockQuantity": 150
}
```

### Redis Commands Used

| Operation | Command | Example |
|-----------|---------|---------|
| Read | GET | `GET products::SKU-001` |
| Write | SET | `SET products::SKU-001 "{...}" EX 60` |
| Delete | DEL | `DEL products::SKU-001` |
| TTL Check | TTL | `TTL products::SKU-001` |

## Error Handling

### ProductNotFoundException

```java
@ResponseStatus(HttpStatus.NOT_FOUND)
public class ProductNotFoundException extends RuntimeException {
    public ProductNotFoundException(String sku) {
        super("Product not found: " + sku);
    }
}
```

### Cache Connection Error

```java
// Handled by Spring's cache error handler
@Bean
public CacheErrorHandler cacheErrorHandler() {
    return new SimpleCacheErrorHandler() {
        @Override
        public void handleCacheGetError(RuntimeException e, Cache cache, Object key) {
            log.warn("Cache get error for key {}: {}", key, e.getMessage());
            // Don't throw - fall back to database
        }
    };
}
```

## Testing Strategy

### Unit Tests

```java
@Test
void getProduct_cacheHit_returnsWithoutDbQuery() {
    // Given: product in cache
    when(redisTemplate.opsForValue().get("products::SKU-001"))
        .thenReturn(productDTO);

    // When
    ProductDTO result = cachedService.getProduct("SKU-001");

    // Then
    verify(productRepository, never()).findBySku(any());
    assertThat(result).isEqualTo(productDTO);
}
```

### Integration Tests

```java
@Test
void getProduct_cacheMiss_loadsFromDbAndCaches() {
    // Given: empty cache, product in DB

    // When: first call
    ProductDTO result1 = cachedService.getProduct("SKU-001");

    // Then: DB was queried
    verify(productRepository, times(1)).findBySku("SKU-001");

    // When: second call
    ProductDTO result2 = cachedService.getProduct("SKU-001");

    // Then: DB was NOT queried again (cache hit)
    verify(productRepository, times(1)).findBySku("SKU-001");
}
```

## Performance Considerations

### Memory Usage

Estimated per cached product:
- Key: ~20 bytes
- Value: ~200 bytes (JSON)
- Overhead: ~50 bytes
- **Total: ~270 bytes per product**

For 10,000 products: ~2.7 MB

### Latency Targets

| Operation | Target | Measurement Point |
|-----------|--------|-------------------|
| Cache Hit | < 2ms | Redis round-trip |
| Cache Miss | < 50ms | DB query + cache write |
| Eviction | < 1ms | Redis DEL |
