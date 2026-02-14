package com.learning.cache.race.service;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import com.learning.cache.common.util.SimulatedDelay;
import com.learning.cache.race.exception.ProductNotFoundException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.redisson.api.RLock;
import org.redisson.api.RedissonClient;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicLong;

@Service
@RequiredArgsConstructor
@Slf4j
public class MutexCacheService {

    private static final String CACHE_PREFIX = "mutex:product:";
    private static final String LOCK_PREFIX = "lock:product:";
    private static final Duration TTL = Duration.ofSeconds(60);

    private final RedisTemplate<String, Object> redisTemplate;
    private final ProductRepository productRepository;
    private final RedissonClient redissonClient;

    @Value("${race-condition.mutex.wait-time:5000}")
    private long lockWaitTimeMs;

    @Value("${race-condition.mutex.lease-time:30000}")
    private long lockLeaseTimeMs;

    // Metrics
    private final AtomicInteger dbQueryCount = new AtomicInteger(0);
    private final AtomicLong totalLockWaitTime = new AtomicLong(0);

    public ProductDTO getProduct(String sku) {
        String cacheKey = CACHE_PREFIX + sku;

        // First check: without lock
        ProductDTO cached = (ProductDTO) redisTemplate.opsForValue().get(cacheKey);
        if (cached != null) {
            log.debug("Cache HIT for {}", sku);
            return cached;
        }

        // Cache miss - need to acquire lock
        String lockKey = LOCK_PREFIX + sku;
        RLock lock = redissonClient.getLock(lockKey);

        long lockStartTime = System.currentTimeMillis();

        try {
            boolean acquired = lock.tryLock(lockWaitTimeMs, lockLeaseTimeMs, TimeUnit.MILLISECONDS);

            if (!acquired) {
                log.warn("Could not acquire lock for {}, attempting cache read", sku);
                // Try one more cache read - maybe another thread filled it
                cached = (ProductDTO) redisTemplate.opsForValue().get(cacheKey);
                if (cached != null) {
                    return cached;
                }
                throw new RuntimeException("Could not acquire lock and cache still empty for: " + sku);
            }

            long lockWaitTime = System.currentTimeMillis() - lockStartTime;
            totalLockWaitTime.addAndGet(lockWaitTime);
            log.debug("Acquired lock for {} after {} ms", sku, lockWaitTime);

            // Double-check after acquiring lock
            cached = (ProductDTO) redisTemplate.opsForValue().get(cacheKey);
            if (cached != null) {
                log.debug("Cache filled by another thread for {}", sku);
                return cached;
            }

            // We have the lock and cache is empty - load from DB
            return loadAndCache(sku, cacheKey);

        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new RuntimeException("Lock acquisition interrupted for: " + sku, e);
        } finally {
            if (lock.isHeldByCurrentThread()) {
                lock.unlock();
                log.debug("Released lock for {}", sku);
            }
        }
    }

    private ProductDTO loadAndCache(String sku, String cacheKey) {
        int queryNumber = dbQueryCount.incrementAndGet();
        log.info("Database query #{} for SKU: {} (mutex protected)", queryNumber, sku);

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
    }

    public void clearAllCache() {
        redisTemplate.delete(redisTemplate.keys(CACHE_PREFIX + "*"));
    }

    public int getAndResetDbQueryCount() {
        return dbQueryCount.getAndSet(0);
    }

    public int getDbQueryCount() {
        return dbQueryCount.get();
    }

    public long getAndResetTotalLockWaitTime() {
        return totalLockWaitTime.getAndSet(0);
    }
}
