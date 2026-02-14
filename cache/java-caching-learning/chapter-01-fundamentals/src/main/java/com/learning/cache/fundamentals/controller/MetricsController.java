package com.learning.cache.fundamentals.controller;

import com.learning.cache.common.metrics.CacheMetrics;
import com.learning.cache.fundamentals.service.CachedProductService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

@RestController
@RequestMapping("/api/metrics")
@RequiredArgsConstructor
@Slf4j
public class MetricsController {

    private final CachedProductService cachedProductService;
    private final CacheMetrics cacheMetrics;

    @GetMapping("/cache")
    public ResponseEntity<CacheMetrics.CacheStats> getCacheStats() {
        log.info("GET /api/metrics/cache");
        return ResponseEntity.ok(cachedProductService.getStats());
    }

    @GetMapping("/cache/all")
    public ResponseEntity<Map<String, CacheMetrics.CacheStats>> getAllCacheStats() {
        log.info("GET /api/metrics/cache/all");

        Map<String, CacheMetrics.CacheStats> stats = Map.of(
                "products", cacheMetrics.getStats("products")
        );

        return ResponseEntity.ok(stats);
    }

    @GetMapping("/health")
    public ResponseEntity<Map<String, Object>> getHealth() {
        CacheMetrics.CacheStats stats = cachedProductService.getStats();

        Map<String, Object> health = Map.of(
                "status", "UP",
                "cache", Map.of(
                        "hitRatio", stats.getHitRatio(),
                        "totalRequests", stats.getTotalRequests(),
                        "healthy", stats.getHitRatio() > 0.8 || stats.getTotalRequests() < 10
                )
        );

        return ResponseEntity.ok(health);
    }
}
