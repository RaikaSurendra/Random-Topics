# Java Caching Learning Curriculum

A structured learning path to master caching concepts from fundamentals to production-ready patterns.

## Prerequisites

Before starting this course, you should be familiar with:
- Java basics (classes, interfaces, generics)
- Spring Boot fundamentals
- Basic SQL and relational databases
- Docker basics
- REST APIs

## Course Structure

Each chapter follows a consistent structure:
1. **Theory** - Understanding the problem and solution
2. **HLD** - High-level architecture and design decisions
3. **LLD** - Detailed class diagrams and method signatures
4. **Implementation** - Hands-on coding
5. **Demonstration** - Live examples and visualizations
6. **Verification** - Testing and metrics validation

---

## Chapter 01: Caching Fundamentals

**Duration**: 2-3 hours

### Learning Objectives
- Understand why caching improves performance
- Implement the cache-aside pattern
- Configure Redis with Spring Boot
- Measure hit/miss ratios
- Use TTL for automatic expiration

### Topics Covered
1. The cache-aside (lazy-loading) pattern
2. Spring's `@Cacheable`, `@CacheEvict`, `@CachePut`
3. Redis as a distributed cache
4. TTL (Time-To-Live) strategies
5. Cache metrics with Micrometer

### Hands-on Exercises
- [ ] Start the application without caching, measure latency
- [ ] Enable caching, observe latency improvement
- [ ] Expire cache entries with TTL
- [ ] Manually evict cache on data updates
- [ ] View hit/miss ratios in Grafana

### Key Takeaways
- Caching reduces database load and improves response time
- TTL prevents stale data but creates temporary inconsistency
- Always measure with metrics, don't guess

---

## Chapter 02: Bloom Filters

**Duration**: 2-3 hours

### Learning Objectives
- Understand cache penetration attacks
- Implement Bloom filters for protection
- Balance false positive rate vs memory
- Use RedisBloom for distributed filters

### Topics Covered
1. Cache penetration problem
2. Bloom filter data structure
3. False positives vs false negatives
4. Guava BloomFilter (in-memory)
5. RedisBloom (distributed)
6. Filter sizing and rebuild strategies

### Hands-on Exercises
- [ ] Simulate penetration attack without protection
- [ ] Add Bloom filter, observe rejection rate
- [ ] Measure false positive rate
- [ ] Compare memory vs accuracy tradeoffs
- [ ] Rebuild filter after data changes

### Key Takeaways
- Bloom filters prevent expensive lookups for non-existent keys
- False positives are acceptable; false negatives are not
- Size the filter based on expected data volume

---

## Chapter 03: Cache Race Conditions

**Duration**: 3-4 hours

### Learning Objectives
- Identify thundering herd problem
- Implement mutex-based solutions
- Use probabilistic early refresh
- Coalesce concurrent requests

### Topics Covered
1. Thundering herd (cache stampede)
2. Dogpile effect
3. Distributed mutex with Redisson
4. Probabilistic refresh (XFetch)
5. Request coalescing/singleflight
6. Comparing solutions

### Hands-on Exercises
- [ ] Simulate 1000 concurrent requests on expired key
- [ ] Observe database overwhelmed (vulnerable version)
- [ ] Implement mutex solution, see single DB query
- [ ] Compare mutex vs probabilistic refresh
- [ ] Implement request coalescing

### Key Takeaways
- Expiration creates coordination problems
- Mutex ensures single writer but adds latency
- Probabilistic refresh prevents expiration spikes
- Choose based on your consistency vs latency needs

---

## Chapter 04: Distributed Caching

**Duration**: 3-4 hours

### Learning Objectives
- Distribute cache across multiple nodes
- Implement consistent hashing
- Handle node failures gracefully
- Use circuit breakers for resilience

### Topics Covered
1. Why distribute cache?
2. Consistent hashing algorithm
3. Virtual nodes for better distribution
4. Handling node additions/removals
5. Circuit breaker pattern
6. Fallback strategies

### Hands-on Exercises
- [ ] Visualize key distribution across nodes
- [ ] Remove a node, observe key redistribution
- [ ] Simulate node failure, see circuit breaker activate
- [ ] Implement local cache fallback
- [ ] Measure impact of node failure

### Key Takeaways
- Consistent hashing minimizes redistribution
- Virtual nodes improve load balance
- Circuit breakers prevent cascade failures
- Always have a fallback strategy

---

## Chapter 05: Cache Consistency

**Duration**: 4-5 hours

### Learning Objectives
- Understand consistency vs availability tradeoffs
- Implement event-driven invalidation
- Compare write patterns (through, behind, ahead)
- Measure inconsistency windows

### Topics Covered
1. CAP theorem implications
2. RabbitMQ for cache invalidation
3. Write-through pattern
4. Write-behind (async batching)
5. Refresh-ahead pattern
6. Eventual consistency measurement

### Hands-on Exercises
- [ ] Observe inconsistency without invalidation events
- [ ] Add RabbitMQ invalidation, measure propagation time
- [ ] Implement write-through for strong consistency
- [ ] Implement write-behind for better write performance
- [ ] Measure inconsistency window duration

### Key Takeaways
- Strong consistency costs performance
- Event-driven invalidation scales better than polling
- Write-behind improves throughput but risks data loss
- Measure your actual inconsistency window

---

## Chapter 06: NGINX Caching

**Duration**: 2-3 hours

### Learning Objectives
- Add HTTP caching layer with NGINX
- Configure Cache-Control headers
- Implement cache purging
- Use microcaching for dynamic content

### Topics Covered
1. HTTP caching fundamentals
2. Cache-Control header directives
3. NGINX proxy_cache configuration
4. X-Cache-Status header monitoring
5. Microcaching strategy
6. Cache purge API

### Hands-on Exercises
- [ ] Configure NGINX proxy_cache
- [ ] Monitor X-Cache-Status (HIT/MISS/STALE)
- [ ] Set appropriate Cache-Control headers
- [ ] Implement microcaching (1-second TTL)
- [ ] Purge cache and observe re-population

### Key Takeaways
- NGINX caching offloads application servers
- Proper headers ensure correct caching behavior
- Microcaching helps even for "dynamic" content
- proxy_cache_lock prevents stampede at NGINX level

---

## Chapter 07: Read Replicas

**Duration**: 3-4 hours

### Learning Objectives
- Implement two-level caching (L1/L2)
- Use Caffeine for local caching
- Configure Redis read replicas
- Handle replica lag

### Topics Covered
1. L1 (local) vs L2 (distributed) caching
2. Caffeine configuration and eviction
3. Redis master-replica topology
4. Read distribution strategies
5. Near-cache pattern
6. Replica lag considerations

### Hands-on Exercises
- [ ] Configure Caffeine as L1 cache
- [ ] Measure L1 hit latency (< 100μs)
- [ ] Compare with L2 latency (< 5ms)
- [ ] Implement tiered lookup
- [ ] Handle L1 invalidation across nodes
- [ ] Measure replica lag

### Key Takeaways
- Local cache provides microsecond access
- Tiered caching maximizes efficiency
- L1 invalidation is challenging in distributed systems
- Consider replica lag for read-after-write scenarios

---

## Chapter 08: Advanced Patterns

**Duration**: 4-5 hours

### Learning Objectives
- Warm cache on startup
- Detect and mitigate hot keys
- Prevent synchronized expiration
- Version cache for zero-downtime deployments

### Topics Covered
1. Cache warming strategies
2. Hot key detection algorithms
3. Hot key mitigation (replication, local cache)
4. TTL jitter for distributed expiration
5. Cache versioning for schema changes
6. Zero-downtime deployment patterns

### Hands-on Exercises
- [ ] Implement startup cache warming
- [ ] Detect hot keys from access patterns
- [ ] Automatically replicate hot keys locally
- [ ] Add TTL jitter, observe smoother expiration
- [ ] Deploy with version switch, verify zero downtime

### Key Takeaways
- Warming reduces cold-start latency
- Hot keys need special handling
- TTL jitter prevents synchronized thundering
- Version prefixes enable safe migrations

---

## Assessment Checklist

After completing all chapters, you should be able to:

- [ ] Explain cache-aside, write-through, write-behind patterns
- [ ] Choose appropriate TTL values for different use cases
- [ ] Protect against cache penetration with Bloom filters
- [ ] Prevent thundering herd with mutex or probabilistic refresh
- [ ] Distribute cache with consistent hashing
- [ ] Handle failures with circuit breakers
- [ ] Implement event-driven cache invalidation
- [ ] Configure NGINX as caching proxy
- [ ] Build two-level cache architecture
- [ ] Detect and mitigate hot keys
- [ ] Deploy cache changes with zero downtime

## Next Steps

After mastering these fundamentals:

1. **Production Deployment**
   - Redis Cluster for high availability
   - Sentinel for automatic failover
   - Monitoring and alerting

2. **Advanced Topics**
   - Cache compression
   - Custom serialization
   - Multi-region caching
   - Cache cost optimization

3. **Real-World Practice**
   - Apply patterns to your projects
   - Conduct load testing
   - Measure and optimize

## Resources

- [Redis Documentation](https://redis.io/docs/)
- [Spring Cache Abstraction](https://docs.spring.io/spring-framework/reference/integration/cache.html)
- [Caffeine Wiki](https://github.com/ben-manes/caffeine/wiki)
- [NGINX Caching Guide](https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_cache)
