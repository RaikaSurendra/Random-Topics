package com.learning.cache.common.metrics;

import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import io.micrometer.core.instrument.Timer;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.TimeUnit;
import java.util.function.Supplier;

@Component
@Slf4j
public class CacheMetrics {

    private final MeterRegistry meterRegistry;
    private final Map<String, Counter> hitCounters = new ConcurrentHashMap<>();
    private final Map<String, Counter> missCounters = new ConcurrentHashMap<>();
    private final Map<String, Counter> evictionCounters = new ConcurrentHashMap<>();
    private final Map<String, Timer> latencyTimers = new ConcurrentHashMap<>();

    public CacheMetrics(MeterRegistry meterRegistry) {
        this.meterRegistry = meterRegistry;
    }

    public void recordHit(String cacheName) {
        getHitCounter(cacheName).increment();
        log.debug("Cache HIT for cache: {}", cacheName);
    }

    public void recordMiss(String cacheName) {
        getMissCounter(cacheName).increment();
        log.debug("Cache MISS for cache: {}", cacheName);
    }

    public void recordEviction(String cacheName) {
        getEvictionCounter(cacheName).increment();
        log.debug("Cache EVICTION for cache: {}", cacheName);
    }

    public void recordLatency(String cacheName, String operation, long durationNanos) {
        getLatencyTimer(cacheName, operation).record(durationNanos, TimeUnit.NANOSECONDS);
    }

    public <T> T recordLatency(String cacheName, String operation, Supplier<T> action) {
        long start = System.nanoTime();
        try {
            return action.get();
        } finally {
            recordLatency(cacheName, operation, System.nanoTime() - start);
        }
    }

    public double getHitRatio(String cacheName) {
        double hits = getHitCounter(cacheName).count();
        double misses = getMissCounter(cacheName).count();
        double total = hits + misses;
        return total == 0 ? 0.0 : hits / total;
    }

    public CacheStats getStats(String cacheName) {
        return CacheStats.builder()
                .cacheName(cacheName)
                .hits((long) getHitCounter(cacheName).count())
                .misses((long) getMissCounter(cacheName).count())
                .evictions((long) getEvictionCounter(cacheName).count())
                .hitRatio(getHitRatio(cacheName))
                .build();
    }

    private Counter getHitCounter(String cacheName) {
        return hitCounters.computeIfAbsent(cacheName, name ->
                Counter.builder("cache.hits")
                        .tag("cache", name)
                        .description("Number of cache hits")
                        .register(meterRegistry));
    }

    private Counter getMissCounter(String cacheName) {
        return missCounters.computeIfAbsent(cacheName, name ->
                Counter.builder("cache.misses")
                        .tag("cache", name)
                        .description("Number of cache misses")
                        .register(meterRegistry));
    }

    private Counter getEvictionCounter(String cacheName) {
        return evictionCounters.computeIfAbsent(cacheName, name ->
                Counter.builder("cache.evictions")
                        .tag("cache", name)
                        .description("Number of cache evictions")
                        .register(meterRegistry));
    }

    private Timer getLatencyTimer(String cacheName, String operation) {
        String key = cacheName + ":" + operation;
        return latencyTimers.computeIfAbsent(key, k ->
                Timer.builder("cache.latency")
                        .tag("cache", cacheName)
                        .tag("operation", operation)
                        .description("Cache operation latency")
                        .register(meterRegistry));
    }

    @lombok.Builder
    @lombok.Data
    public static class CacheStats {
        private String cacheName;
        private long hits;
        private long misses;
        private long evictions;
        private double hitRatio;

        public long getTotalRequests() {
            return hits + misses;
        }

        public String getHitRatioPercentage() {
            return String.format("%.2f%%", hitRatio * 100);
        }
    }
}
