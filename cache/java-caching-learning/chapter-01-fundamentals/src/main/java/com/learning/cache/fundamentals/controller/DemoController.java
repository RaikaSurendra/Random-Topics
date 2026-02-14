package com.learning.cache.fundamentals.controller;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.metrics.CacheMetrics;
import com.learning.cache.common.util.LoadGenerator;
import com.learning.cache.fundamentals.service.CachedProductService;
import com.learning.cache.fundamentals.service.ProductService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/demo")
@RequiredArgsConstructor
@Slf4j
public class DemoController {

    private final CachedProductService cachedProductService;
    private final ProductService productService;

    @GetMapping("/compare/{sku}")
    public ResponseEntity<Map<String, Object>> compareCachedVsUncached(@PathVariable String sku) {
        log.info("Running cached vs uncached comparison for SKU: {}", sku);

        // Warm up cache
        cachedProductService.getProduct(sku);

        // Measure uncached
        long uncachedStart = System.nanoTime();
        productService.getProduct(sku);
        long uncachedDuration = System.nanoTime() - uncachedStart;

        // Measure cached
        long cachedStart = System.nanoTime();
        cachedProductService.getProduct(sku);
        long cachedDuration = System.nanoTime() - cachedStart;

        double speedup = (double) uncachedDuration / cachedDuration;

        Map<String, Object> result = Map.of(
                "sku", sku,
                "uncachedLatencyMs", uncachedDuration / 1_000_000.0,
                "cachedLatencyMs", cachedDuration / 1_000_000.0,
                "speedupFactor", String.format("%.2fx", speedup),
                "conclusion", speedup > 5 ? "Cache is significantly faster!" : "Cache provides modest improvement"
        );

        return ResponseEntity.ok(result);
    }

    @PostMapping("/load-test")
    public ResponseEntity<Map<String, Object>> runLoadTest(
            @RequestParam(defaultValue = "100") int requests,
            @RequestParam(defaultValue = "SKU-001") String sku,
            @RequestParam(defaultValue = "true") boolean useCaching) {

        log.info("Running load test: {} requests for SKU {} (caching: {})",
                requests, sku, useCaching);

        // Clear cache and stats for clean test
        cachedProductService.clearCache();

        LoadGenerator loadGenerator = new LoadGenerator(20);

        LoadGenerator.LoadResult<ProductDTO> result;
        if (useCaching) {
            result = loadGenerator.runConcurrent(requests,
                    () -> cachedProductService.getProduct(sku));
        } else {
            result = loadGenerator.runConcurrent(requests,
                    () -> productService.getProduct(sku));
        }

        loadGenerator.shutdown();

        CacheMetrics.CacheStats stats = cachedProductService.getStats();

        Map<String, Object> response = new HashMap<>();
        response.put("requests", requests);
        response.put("useCaching", useCaching);
        response.put("successRate", String.format("%.2f%%", result.getSuccessRate() * 100));
        response.put("throughputRps", String.format("%.2f", result.getRequestsPerSecond()));
        response.put("avgLatencyMs", String.format("%.2f", result.getAvgLatencyNanos() / 1_000_000.0));
        response.put("p99LatencyMs", String.format("%.2f", result.getP99LatencyNanos() / 1_000_000.0));

        if (useCaching) {
            response.put("cacheHits", stats.getHits());
            response.put("cacheMisses", stats.getMisses());
            response.put("hitRatio", stats.getHitRatioPercentage());
        }

        return ResponseEntity.ok(response);
    }

    @GetMapping("/ttl-demo/{sku}")
    public ResponseEntity<Map<String, Object>> demonstrateTtl(@PathVariable String sku) {
        log.info("Demonstrating TTL behavior for SKU: {}", sku);

        // Initial load (cache miss)
        long start1 = System.nanoTime();
        cachedProductService.getProduct(sku);
        long duration1 = System.nanoTime() - start1;

        // Immediate second call (cache hit)
        long start2 = System.nanoTime();
        cachedProductService.getProduct(sku);
        long duration2 = System.nanoTime() - start2;

        CacheMetrics.CacheStats stats = cachedProductService.getStats();

        Map<String, Object> result = Map.of(
                "sku", sku,
                "firstCallMs", duration1 / 1_000_000.0,
                "secondCallMs", duration2 / 1_000_000.0,
                "hitRatio", stats.getHitRatioPercentage(),
                "note", "Wait for TTL (default 60s) to expire and call again to see cache miss"
        );

        return ResponseEntity.ok(result);
    }

    @GetMapping("/all-skus")
    public ResponseEntity<List<String>> getAllSkus() {
        return ResponseEntity.ok(productService.getAllSkus());
    }
}
