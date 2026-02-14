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
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicInteger;

@Service
@RequiredArgsConstructor
@Slf4j
public class CoalescingCacheService {

    private static final String CACHE_PREFIX = "coalesce:product:";
    private static final Duration TTL = Duration.ofSeconds(60);

    private final RedisTemplate<String, Object> redisTemplate;
    private final ProductRepository productRepository;

    @Value("${race-condition.coalescing.timeout:10000}")
    private long coalescingTimeoutMs;

    // In-flight requests map - key -> Future
    // All concurrent requests for the same key will share the same Future
    private final ConcurrentHashMap<String, CompletableFuture<ProductDTO>> inFlight = new ConcurrentHashMap<>();

    // Metrics
    private final AtomicInteger dbQueryCount = new AtomicInteger(0);
    private final AtomicInteger coalescedCount = new AtomicInteger(0);

    public ProductDTO getProduct(String sku) {
        String cacheKey = CACHE_PREFIX + sku;

        // Check cache first
        ProductDTO cached = (ProductDTO) redisTemplate.opsForValue().get(cacheKey);
        if (cached != null) {
            log.debug("Cache HIT for {}", sku);
            return cached;
        }

        // Cache miss - use singleflight pattern
        log.debug("Cache MISS for {} - checking for in-flight request", sku);

        CompletableFuture<ProductDTO> future = inFlight.computeIfAbsent(sku, key -> {
            log.info("Creating new in-flight request for {}", key);
            return loadAsync(key);
        });

        // If we didn't create the future, we coalesced with existing request
        if (inFlight.get(sku) != future) {
            coalescedCount.incrementAndGet();
            log.debug("Coalesced request for {}", sku);
        }

        try {
            return future.get(coalescingTimeoutMs, TimeUnit.MILLISECONDS);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new RuntimeException("Request interrupted for: " + sku, e);
        } catch (ExecutionException e) {
            Throwable cause = e.getCause();
            if (cause instanceof ProductNotFoundException) {
                throw (ProductNotFoundException) cause;
            }
            throw new RuntimeException("Failed to load product: " + sku, cause);
        } catch (TimeoutException e) {
            throw new RuntimeException("Timeout waiting for product: " + sku, e);
        }
    }

    private CompletableFuture<ProductDTO> loadAsync(String sku) {
        return CompletableFuture.supplyAsync(() -> {
            try {
                return loadAndCache(sku);
            } finally {
                // Remove from in-flight map after completion
                inFlight.remove(sku);
            }
        });
    }

    private ProductDTO loadAndCache(String sku) {
        String cacheKey = CACHE_PREFIX + sku;

        int queryNumber = dbQueryCount.incrementAndGet();
        log.info("Database query #{} for SKU: {} (coalescing)", queryNumber, sku);

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

    public int getAndResetCoalescedCount() {
        return coalescedCount.getAndSet(0);
    }

    public int getCoalescedCount() {
        return coalescedCount.get();
    }

    public int getInFlightCount() {
        return inFlight.size();
    }
}
