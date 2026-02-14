package com.learning.cache.bloom.controller;

import com.learning.cache.bloom.demo.PenetrationAttackDemo;
import com.learning.cache.bloom.metrics.BloomMetrics;
import com.learning.cache.bloom.service.BloomProtectedService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

@RestController
@RequestMapping("/api/bloom")
@RequiredArgsConstructor
@Slf4j
public class BloomDemoController {

    private final BloomProtectedService bloomProtectedService;
    private final PenetrationAttackDemo attackDemo;

    @GetMapping("/stats")
    public ResponseEntity<BloomMetrics.BloomStats> getStats() {
        log.info("GET /api/bloom/stats");
        return ResponseEntity.ok(bloomProtectedService.getStats());
    }

    @PostMapping("/rebuild")
    public ResponseEntity<Map<String, Object>> rebuildFilter() {
        log.info("POST /api/bloom/rebuild");
        long start = System.currentTimeMillis();

        bloomProtectedService.rebuildFilter();

        long duration = System.currentTimeMillis() - start;
        BloomMetrics.BloomStats stats = bloomProtectedService.getStats();

        return ResponseEntity.ok(Map.of(
                "status", "success",
                "rebuildTimeMs", duration,
                "approximateSize", stats.getApproximateSize()
        ));
    }

    @PostMapping("/demo/attack")
    public ResponseEntity<PenetrationAttackDemo.AttackResult> runAttack(
            @RequestParam(defaultValue = "1000") int requests,
            @RequestParam(defaultValue = "true") boolean useProtection) {
        log.info("POST /api/bloom/demo/attack - requests={}, protected={}", requests, useProtection);
        return ResponseEntity.ok(attackDemo.runAttack(requests, useProtection));
    }

    @GetMapping("/demo/false-positives")
    public ResponseEntity<PenetrationAttackDemo.FalsePositiveResult> measureFalsePositives(
            @RequestParam(defaultValue = "10000") int tests) {
        log.info("GET /api/bloom/demo/false-positives - tests={}", tests);
        return ResponseEntity.ok(attackDemo.measureFalsePositives(tests));
    }

    @PostMapping("/demo/compare")
    public ResponseEntity<Map<String, Object>> compareProtectedVsUnprotected(
            @RequestParam(defaultValue = "500") int requests) {
        log.info("POST /api/bloom/demo/compare - requests={}", requests);

        PenetrationAttackDemo.AttackResult unprotected = attackDemo.runAttack(requests, false);
        PenetrationAttackDemo.AttackResult protectedResult = attackDemo.runAttack(requests, true);

        double improvementFactor = unprotected.getAvgLatencyMs() / Math.max(0.001, protectedResult.getAvgLatencyMs());

        return ResponseEntity.ok(Map.of(
                "requests", requests,
                "unprotected", unprotected,
                "protected", protectedResult,
                "improvementFactor", String.format("%.2fx faster", improvementFactor),
                "dbQueriesAvoided", unprotected.getReachedDatabase() - protectedResult.getReachedDatabase()
        ));
    }
}
