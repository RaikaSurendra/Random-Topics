package com.learning.cache.bloom.demo;

import com.learning.cache.bloom.exception.ProductNotFoundException;
import com.learning.cache.bloom.service.BloomFilterService;
import com.learning.cache.bloom.service.BloomProtectedService;
import com.learning.cache.common.util.LoadGenerator;
import lombok.Builder;
import lombok.Data;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.stream.IntStream;

@Component
@RequiredArgsConstructor
@Slf4j
public class PenetrationAttackDemo {

    private final BloomProtectedService bloomProtectedService;
    private final BloomFilterService bloomFilter;

    public AttackResult runAttack(int requestCount, boolean useProtection) {
        log.info("Running penetration attack: {} requests, protection={}", requestCount, useProtection);

        // Generate fake SKUs that don't exist in the database
        List<String> fakeSKUs = generateFakeSkus(requestCount);

        AtomicInteger blockedByBloom = new AtomicInteger(0);
        AtomicInteger reachedDatabase = new AtomicInteger(0);
        AtomicInteger errors = new AtomicInteger(0);

        LoadGenerator loadGenerator = new LoadGenerator(20);

        long start = System.currentTimeMillis();

        LoadGenerator.LoadResult<String> result = loadGenerator.runConcurrent(requestCount, () -> {
            String fakeSku = fakeSKUs.get((int) (Math.random() * fakeSKUs.size()));

            try {
                if (useProtection) {
                    // This will check Bloom filter first
                    bloomProtectedService.getProduct(fakeSku);
                } else {
                    // This bypasses Bloom filter
                    bloomProtectedService.getProductUnprotected(fakeSku);
                }
                // If we get here, it was a false positive (shouldn't happen with fake SKUs)
                return "unexpected_success";
            } catch (ProductNotFoundException e) {
                // Expected - the SKU doesn't exist
                if (useProtection && !bloomFilter.mightContain(fakeSku)) {
                    blockedByBloom.incrementAndGet();
                } else {
                    reachedDatabase.incrementAndGet();
                }
                return "not_found";
            } catch (Exception e) {
                errors.incrementAndGet();
                return "error";
            }
        });

        loadGenerator.shutdown();

        long duration = System.currentTimeMillis() - start;

        AttackResult attackResult = AttackResult.builder()
                .totalRequests(requestCount)
                .useProtection(useProtection)
                .blockedByBloom(blockedByBloom.get())
                .reachedDatabase(reachedDatabase.get())
                .errors(errors.get())
                .totalDurationMs(duration)
                .avgLatencyMs(result.getAvgLatencyNanos() / 1_000_000.0)
                .p99LatencyMs(result.getP99LatencyNanos() / 1_000_000.0)
                .requestsPerSecond(result.getRequestsPerSecond())
                .build();

        log.info("Attack result: {}", attackResult);
        return attackResult;
    }

    public FalsePositiveResult measureFalsePositives(int testCount) {
        log.info("Measuring false positive rate with {} tests", testCount);

        int falsePositives = 0;

        for (int i = 0; i < testCount; i++) {
            String fakeSku = "FAKE-" + UUID.randomUUID();
            if (bloomFilter.mightContain(fakeSku)) {
                falsePositives++;
            }
        }

        double actualRate = (double) falsePositives / testCount;
        double configuredRate = bloomFilter.expectedFalsePositiveRate();

        FalsePositiveResult result = FalsePositiveResult.builder()
                .tests(testCount)
                .falsePositives(falsePositives)
                .actualRate(actualRate)
                .configuredRate(configuredRate)
                .withinBounds(actualRate <= configuredRate * 1.5) // Allow 50% margin
                .build();

        log.info("False positive measurement: {}", result);
        return result;
    }

    private List<String> generateFakeSkus(int count) {
        return IntStream.range(0, Math.min(count, 10000))
                .mapToObj(i -> "FAKE-SKU-" + UUID.randomUUID().toString().substring(0, 8))
                .toList();
    }

    @Data
    @Builder
    public static class AttackResult {
        private int totalRequests;
        private boolean useProtection;
        private int blockedByBloom;
        private int reachedDatabase;
        private int errors;
        private long totalDurationMs;
        private double avgLatencyMs;
        private double p99LatencyMs;
        private double requestsPerSecond;

        public String getBlockRate() {
            if (!useProtection) return "N/A";
            return String.format("%.2f%%", (double) blockedByBloom / totalRequests * 100);
        }

        public String getConclusion() {
            if (!useProtection) {
                return "All " + reachedDatabase + " requests hit the database!";
            }
            double blockRate = (double) blockedByBloom / totalRequests * 100;
            if (blockRate > 99) {
                return "Excellent! " + String.format("%.2f%%", blockRate) + " of attacks blocked by Bloom filter.";
            } else if (blockRate > 90) {
                return "Good. " + String.format("%.2f%%", blockRate) + " blocked. Consider tuning false positive rate.";
            } else {
                return "Warning: Only " + String.format("%.2f%%", blockRate) + " blocked. Rebuild filter may be needed.";
            }
        }
    }

    @Data
    @Builder
    public static class FalsePositiveResult {
        private int tests;
        private int falsePositives;
        private double actualRate;
        private double configuredRate;
        private boolean withinBounds;

        public String getActualRatePercentage() {
            return String.format("%.4f%%", actualRate * 100);
        }

        public String getConfiguredRatePercentage() {
            return String.format("%.2f%%", configuredRate * 100);
        }

        public String getConclusion() {
            if (withinBounds) {
                return "False positive rate is within expected bounds.";
            } else {
                return "Warning: False positive rate exceeds configured threshold. Consider rebuilding filter.";
            }
        }
    }
}
