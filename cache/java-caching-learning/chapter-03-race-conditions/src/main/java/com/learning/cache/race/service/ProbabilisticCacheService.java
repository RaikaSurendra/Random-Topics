package com.learning.cache.race.service;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import com.learning.cache.common.util.SimulatedDelay;
import com.learning.cache.race.exception.ProductNotFoundException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.scheduling.annotation.Async;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

@Service
@RequiredArgsConstructor
@Slf4j
public class ProbabilisticCacheService {

    private static final String CACHE_PREFIX = "prob:product:";
    private static final String EXPIRY_PREFIX = "prob:expiry:";
    private static final Duration TTL = Duration.ofSeconds(60);

    private final RedisTemplate<String, Object> redisTemplate;
    private final ProductRepository productRepository;

    @Value("${race-condition.probabilistic.beta:1.0}")
    private double beta; // Controls refresh aggressiveness

    @Value("${race-condition.probabilistic.delta:5000}")
    private long deltaMs; // Refresh window before expiration

    // Track which keys are being refreshed to prevent duplicate refreshes
    private final Set<String> refreshingKeys = ConcurrentHashMap.newKeySet();

    // Metrics
    private final AtomicInteger dbQueryCount = new AtomicInteger(0);
    private final AtomicInteger backgroundRefreshCount = new AtomicInteger(0);

    public ProductDTO getProduct(String sku) {
        String cacheKey = CACHE_PREFIX + sku;
        String expiryKey = EXPIRY_PREFIX + sku;

        ProductDTO cached = (ProductDTO) redisTemplate.opsForValue().get(cacheKey);

        if (cached != null) {
            // Check if we should proactively refresh
            Long expiryTime = (Long) redisTemplate.opsForValue().get(expiryKey);
            if (expiryTime != null) {
                long now = System.currentTimeMillis();
                long remainingTtl = expiryTime - now;

                if (shouldRefresh(remainingTtl)) {
                    log.debug("Probabilistic refresh triggered for {} (remaining TTL: {} ms)", sku, remainingTtl);
                    asyncRefresh(sku);
                }
            }
            return cached;
        }

        // Cache miss - must load synchronously
        log.debug("Cache MISS for {} - loading from database", sku);
        return loadAndCache(sku);
    }

    private boolean shouldRefresh(long remainingTtlMs) {
        if (remainingTtlMs <= 0) return true;
        if (remainingTtlMs > deltaMs) return false;

        // XFetch algorithm: probability increases as TTL approaches 0
        // P(refresh) = beta * log(delta / remainingTtl)
        double probability = beta * Math.log((double) deltaMs / remainingTtlMs);
        probability = Math.min(1.0, Math.max(0.0, probability));

        boolean shouldRefresh = Math.random() < probability;
        log.debug("Refresh probability: {}, should refresh: {}", probability, shouldRefresh);
        return shouldRefresh;
    }

    @Async
    public void asyncRefresh(String sku) {
        // Prevent duplicate concurrent refreshes
        if (!refreshingKeys.add(sku)) {
            log.debug("Refresh already in progress for {}", sku);
            return;
        }

        try {
            log.info("Background refresh started for {}", sku);
            backgroundRefreshCount.incrementAndGet();
            loadAndCache(sku);
            log.info("Background refresh completed for {}", sku);
        } finally {
            refreshingKeys.remove(sku);
        }
    }

    private ProductDTO loadAndCache(String sku) {
        String cacheKey = CACHE_PREFIX + sku;
        String expiryKey = EXPIRY_PREFIX + sku;

        int queryNumber = dbQueryCount.incrementAndGet();
        log.info("Database query #{} for SKU: {} (probabilistic)", queryNumber, sku);

        SimulatedDelay.databaseQuery(30, 50);

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new ProductNotFoundException(sku));

        ProductDTO dto = ProductDTO.from(product);

        // Store in cache with TTL
        redisTemplate.opsForValue().set(cacheKey, dto, TTL);

        // Store expiration timestamp
        long expiryTime = System.currentTimeMillis() + TTL.toMillis();
        redisTemplate.opsForValue().set(expiryKey, expiryTime, TTL.plusSeconds(10));

        return dto;
    }

    public void clearCache(String sku) {
        redisTemplate.delete(CACHE_PREFIX + sku);
        redisTemplate.delete(EXPIRY_PREFIX + sku);
    }

    public void clearAllCache() {
        redisTemplate.delete(redisTemplate.keys(CACHE_PREFIX + "*"));
        redisTemplate.delete(redisTemplate.keys(EXPIRY_PREFIX + "*"));
    }

    public int getAndResetDbQueryCount() {
        return dbQueryCount.getAndSet(0);
    }

    public int getDbQueryCount() {
        return dbQueryCount.get();
    }

    public int getAndResetBackgroundRefreshCount() {
        return backgroundRefreshCount.getAndSet(0);
    }

    public int getBackgroundRefreshCount() {
        return backgroundRefreshCount.get();
    }
}
