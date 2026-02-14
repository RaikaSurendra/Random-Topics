package com.learning.cache.race.service;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import com.learning.cache.common.util.SimulatedDelay;
import com.learning.cache.race.exception.ProductNotFoundException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.concurrent.atomic.AtomicInteger;

@Service
@RequiredArgsConstructor
@Slf4j
public class VulnerableCacheService {

    private static final String CACHE_PREFIX = "vulnerable:product:";
    private static final Duration TTL = Duration.ofSeconds(5); // Short TTL for demo

    private final RedisTemplate<String, Object> redisTemplate;
    private final ProductRepository productRepository;

    // Counter to track DB queries (for demonstration)
    private final AtomicInteger dbQueryCount = new AtomicInteger(0);

    public ProductDTO getProduct(String sku) {
        String cacheKey = CACHE_PREFIX + sku;

        // Check cache
        ProductDTO cached = (ProductDTO) redisTemplate.opsForValue().get(cacheKey);
        if (cached != null) {
            log.debug("Cache HIT for {}", sku);
            return cached;
        }

        // Cache miss - all concurrent requests will hit the database!
        log.debug("Cache MISS for {} - loading from database", sku);
        return loadAndCache(sku, cacheKey);
    }

    private ProductDTO loadAndCache(String sku, String cacheKey) {
        int queryNumber = dbQueryCount.incrementAndGet();
        log.info("Database query #{} for SKU: {}", queryNumber, sku);

        // Simulate database latency
        SimulatedDelay.databaseQuery(30, 50);

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new ProductNotFoundException(sku));

        ProductDTO dto = ProductDTO.from(product);

        // Store in cache
        redisTemplate.opsForValue().set(cacheKey, dto, TTL);

        return dto;
    }

    public void clearCache(String sku) {
        redisTemplate.delete(CACHE_PREFIX + sku);
        log.info("Cleared cache for {}", sku);
    }

    public void clearAllCache() {
        redisTemplate.delete(redisTemplate.keys(CACHE_PREFIX + "*"));
        log.info("Cleared all vulnerable cache entries");
    }

    public int getAndResetDbQueryCount() {
        return dbQueryCount.getAndSet(0);
    }

    public int getDbQueryCount() {
        return dbQueryCount.get();
    }
}
