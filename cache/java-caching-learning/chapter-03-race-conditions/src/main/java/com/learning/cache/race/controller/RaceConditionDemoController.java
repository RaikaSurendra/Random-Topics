package com.learning.cache.race.controller;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.util.LoadGenerator;
import com.learning.cache.race.service.*;
import lombok.Builder;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.Map;

@RestController
@RequestMapping("/api/demo")
@RequiredArgsConstructor
@Slf4j
public class RaceConditionDemoController {

    private final VulnerableCacheService vulnerableService;
    private final MutexCacheService mutexService;
    private final ProbabilisticCacheService probabilisticService;
    private final CoalescingCacheService coalescingService;

    @PostMapping("/stampede")
    public ResponseEntity<StampedeResult> runStampede(
            @RequestParam(defaultValue = "100") int requests,
            @RequestParam(defaultValue = "vulnerable") String strategy,
            @RequestParam(defaultValue = "SKU-001") String sku) {

        log.info("Running stampede test: {} requests, strategy={}, sku={}", requests, strategy, sku);

        // Clear cache to ensure stampede
        clearCacheForStrategy(strategy, sku);

        // Reset counters
        resetCounters(strategy);

        LoadGenerator loadGenerator = new LoadGenerator(50);

        long start = System.currentTimeMillis();

        LoadGenerator.LoadResult<ProductDTO> result = loadGenerator.runConcurrent(requests, () -> {
            return switch (strategy) {
                case "mutex" -> mutexService.getProduct(sku);
                case "probabilistic" -> probabilisticService.getProduct(sku);
                case "coalescing" -> coalescingService.getProduct(sku);
                default -> vulnerableService.getProduct(sku);
            };
        });

        loadGenerator.shutdown();

        long duration = System.currentTimeMillis() - start;

        StampedeResult stampedeResult = buildResult(strategy, requests, result, duration);

        log.info("Stampede result: {}", stampedeResult);
        return ResponseEntity.ok(stampedeResult);
    }

    @GetMapping("/mutex/{sku}")
    public ResponseEntity<ProductDTO> getWithMutex(@PathVariable String sku) {
        return ResponseEntity.ok(mutexService.getProduct(sku));
    }

    @GetMapping("/probabilistic/{sku}")
    public ResponseEntity<ProductDTO> getWithProbabilistic(@PathVariable String sku) {
        return ResponseEntity.ok(probabilisticService.getProduct(sku));
    }

    @GetMapping("/coalescing/{sku}")
    public ResponseEntity<ProductDTO> getWithCoalescing(@PathVariable String sku) {
        return ResponseEntity.ok(coalescingService.getProduct(sku));
    }

    @GetMapping("/vulnerable/{sku}")
    public ResponseEntity<ProductDTO> getVulnerable(@PathVariable String sku) {
        return ResponseEntity.ok(vulnerableService.getProduct(sku));
    }

    @PostMapping("/compare")
    public ResponseEntity<Map<String, StampedeResult>> compareStrategies(
            @RequestParam(defaultValue = "100") int requests,
            @RequestParam(defaultValue = "SKU-001") String sku) {

        log.info("Comparing all strategies with {} requests for {}", requests, sku);

        Map<String, StampedeResult> results = new HashMap<>();

        for (String strategy : new String[]{"vulnerable", "mutex", "coalescing"}) {
            // Clear cache
            clearCacheForStrategy(strategy, sku);
            resetCounters(strategy);

            LoadGenerator loadGenerator = new LoadGenerator(50);
            long start = System.currentTimeMillis();

            LoadGenerator.LoadResult<ProductDTO> result = loadGenerator.runConcurrent(requests, () -> {
                return switch (strategy) {
                    case "mutex" -> mutexService.getProduct(sku);
                    case "coalescing" -> coalescingService.getProduct(sku);
                    default -> vulnerableService.getProduct(sku);
                };
            });

            loadGenerator.shutdown();
            long duration = System.currentTimeMillis() - start;

            results.put(strategy, buildResult(strategy, requests, result, duration));

            // Small delay between tests
            try { Thread.sleep(500); } catch (InterruptedException ignored) {}
        }

        return ResponseEntity.ok(results);
    }

    @DeleteMapping("/cache")
    public ResponseEntity<Void> clearAllCaches() {
        vulnerableService.clearAllCache();
        mutexService.clearAllCache();
        probabilisticService.clearAllCache();
        coalescingService.clearAllCache();
        return ResponseEntity.noContent().build();
    }

    private void clearCacheForStrategy(String strategy, String sku) {
        switch (strategy) {
            case "mutex" -> mutexService.clearCache(sku);
            case "probabilistic" -> probabilisticService.clearCache(sku);
            case "coalescing" -> coalescingService.clearCache(sku);
            default -> vulnerableService.clearCache(sku);
        }
    }

    private void resetCounters(String strategy) {
        switch (strategy) {
            case "mutex" -> {
                mutexService.getAndResetDbQueryCount();
                mutexService.getAndResetTotalLockWaitTime();
            }
            case "probabilistic" -> {
                probabilisticService.getAndResetDbQueryCount();
                probabilisticService.getAndResetBackgroundRefreshCount();
            }
            case "coalescing" -> {
                coalescingService.getAndResetDbQueryCount();
                coalescingService.getAndResetCoalescedCount();
            }
            default -> vulnerableService.getAndResetDbQueryCount();
        }
    }

    private StampedeResult buildResult(String strategy, int requests,
                                        LoadGenerator.LoadResult<ProductDTO> result, long duration) {
        StampedeResult.StampedeResultBuilder builder = StampedeResult.builder()
                .strategy(strategy)
                .requests(requests)
                .successCount(result.getSuccessCount())
                .errorCount(result.getErrorCount())
                .totalDurationMs(duration)
                .avgLatencyMs(result.getAvgLatencyNanos() / 1_000_000.0)
                .p99LatencyMs(result.getP99LatencyNanos() / 1_000_000.0)
                .requestsPerSecond(result.getRequestsPerSecond());

        switch (strategy) {
            case "mutex" -> {
                builder.dbQueries(mutexService.getDbQueryCount());
                builder.lockWaitTimeMs(mutexService.getAndResetTotalLockWaitTime());
            }
            case "probabilistic" -> {
                builder.dbQueries(probabilisticService.getDbQueryCount());
                builder.backgroundRefreshes(probabilisticService.getBackgroundRefreshCount());
            }
            case "coalescing" -> {
                builder.dbQueries(coalescingService.getDbQueryCount());
                builder.coalescedRequests(coalescingService.getCoalescedCount());
            }
            default -> builder.dbQueries(vulnerableService.getDbQueryCount());
        }

        return builder.build();
    }

    @Data
    @Builder
    public static class StampedeResult {
        private String strategy;
        private int requests;
        private int successCount;
        private int errorCount;
        private int dbQueries;
        private long totalDurationMs;
        private double avgLatencyMs;
        private double p99LatencyMs;
        private double requestsPerSecond;

        // Strategy-specific metrics
        private Long lockWaitTimeMs;
        private Integer backgroundRefreshes;
        private Integer coalescedRequests;

        public String getConclusion() {
            if ("vulnerable".equals(strategy)) {
                return String.format("Database hit %d times! Stampede not protected.", dbQueries);
            } else if (dbQueries == 1) {
                return String.format("Only 1 database query for %d requests. Stampede prevented!", requests);
            } else {
                return String.format("%d database queries for %d requests.", dbQueries, requests);
            }
        }

        public double getDbQueryReduction() {
            return requests > 0 ? (1.0 - (double) dbQueries / requests) * 100 : 0;
        }
    }
}
