package com.learning.cache.replicas.service;

import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import com.learning.cache.common.util.SimulatedDelay;
import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Timer;
import jakarta.annotation.PostConstruct;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.concurrent.TimeUnit;

@Service
@RequiredArgsConstructor
@Slf4j
public class TwoLevelCache {

    private static final String L2_PREFIX = "l2:product:";
    private static final Duration L2_TTL = Duration.ofMinutes(5);

    private final RedisTemplate<String, Object> redisTemplate;
    private final ProductRepository productRepository;
    private final MeterRegistry meterRegistry;

    private Cache<String, ProductDTO> l1Cache;

    private Counter l1HitCounter;
    private Counter l1MissCounter;
    private Counter l2HitCounter;
    private Counter l2MissCounter;
    private Timer l1Timer;
    private Timer l2Timer;
    private Timer dbTimer;

    @PostConstruct
    public void initialize() {
        l1Cache = Caffeine.newBuilder()
                .maximumSize(10_000)
                .expireAfterWrite(Duration.ofSeconds(60))
                .recordStats()
                .build();

        l1HitCounter = Counter.builder("cache.l1.hits").register(meterRegistry);
        l1MissCounter = Counter.builder("cache.l1.misses").register(meterRegistry);
        l2HitCounter = Counter.builder("cache.l2.hits").register(meterRegistry);
        l2MissCounter = Counter.builder("cache.l2.misses").register(meterRegistry);
        l1Timer = Timer.builder("cache.l1.latency").register(meterRegistry);
        l2Timer = Timer.builder("cache.l2.latency").register(meterRegistry);
        dbTimer = Timer.builder("cache.db.latency").register(meterRegistry);
    }

    public ProductDTO getProduct(String sku) {
        // L1 Check (Caffeine - in-memory)
        long l1Start = System.nanoTime();
        ProductDTO cached = l1Cache.getIfPresent(sku);
        l1Timer.record(System.nanoTime() - l1Start, TimeUnit.NANOSECONDS);

        if (cached != null) {
            l1HitCounter.increment();
            log.debug("L1 HIT for {}", sku);
            return cached;
        }
        l1MissCounter.increment();
        log.debug("L1 MISS for {}", sku);

        // L2 Check (Redis - distributed)
        String l2Key = L2_PREFIX + sku;
        long l2Start = System.nanoTime();
        cached = (ProductDTO) redisTemplate.opsForValue().get(l2Key);
        l2Timer.record(System.nanoTime() - l2Start, TimeUnit.NANOSECONDS);

        if (cached != null) {
            l2HitCounter.increment();
            log.debug("L2 HIT for {}", sku);
            // Populate L1
            l1Cache.put(sku, cached);
            return cached;
        }
        l2MissCounter.increment();
        log.debug("L2 MISS for {}", sku);

        // Database load
        long dbStart = System.nanoTime();
        SimulatedDelay.databaseQuery();
        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new RuntimeException("Product not found: " + sku));
        ProductDTO dto = ProductDTO.from(product);
        dbTimer.record(System.nanoTime() - dbStart, TimeUnit.NANOSECONDS);

        // Populate L2 and L1
        redisTemplate.opsForValue().set(l2Key, dto, L2_TTL);
        l1Cache.put(sku, dto);

        return dto;
    }

    public void evict(String sku) {
        l1Cache.invalidate(sku);
        redisTemplate.delete(L2_PREFIX + sku);
        log.info("Evicted {} from L1 and L2", sku);
    }

    public void evictL1(String sku) {
        l1Cache.invalidate(sku);
        log.debug("Evicted {} from L1 only", sku);
    }

    public CacheStats getStats() {
        com.github.benmanes.caffeine.cache.stats.CacheStats caffeineStats = l1Cache.stats();
        return new CacheStats(
                (long) l1HitCounter.count(),
                (long) l1MissCounter.count(),
                (long) l2HitCounter.count(),
                (long) l2MissCounter.count(),
                l1Cache.estimatedSize(),
                caffeineStats.hitRate()
        );
    }

    public record CacheStats(
            long l1Hits, long l1Misses,
            long l2Hits, long l2Misses,
            long l1Size, double l1HitRate
    ) {
        public double getOverallHitRate() {
            long totalHits = l1Hits + l2Hits;
            long totalRequests = l1Hits + l1Misses;
            return totalRequests > 0 ? (double) totalHits / totalRequests : 0;
        }
    }
}
