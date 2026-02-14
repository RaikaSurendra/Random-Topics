package com.learning.cache.bloom.metrics;

import io.micrometer.core.instrument.Counter;
import io.micrometer.core.instrument.MeterRegistry;
import lombok.Builder;
import lombok.Data;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

@Component
@Slf4j
public class BloomMetrics {

    private final Counter checkCounter;
    private final Counter rejectionCounter;
    private final Counter falsePositiveCounter;

    public BloomMetrics(MeterRegistry meterRegistry) {
        this.checkCounter = Counter.builder("bloom.checks")
                .description("Total Bloom filter checks")
                .register(meterRegistry);

        this.rejectionCounter = Counter.builder("bloom.rejections")
                .description("Requests rejected by Bloom filter")
                .register(meterRegistry);

        this.falsePositiveCounter = Counter.builder("bloom.false_positives")
                .description("False positives (passed Bloom but not in DB)")
                .register(meterRegistry);
    }

    public void recordCheck() {
        checkCounter.increment();
    }

    public void recordRejection() {
        rejectionCounter.increment();
    }

    public void recordFalsePositive() {
        falsePositiveCounter.increment();
    }

    public BloomStats getStats(long approximateSize, double configuredFalsePositiveRate) {
        double checks = checkCounter.count();
        double rejections = rejectionCounter.count();
        double falsePositives = falsePositiveCounter.count();

        double rejectionRate = checks > 0 ? rejections / checks : 0;
        double actualFalsePositiveRate = (checks - rejections) > 0
                ? falsePositives / (checks - rejections) : 0;

        return BloomStats.builder()
                .totalChecks((long) checks)
                .rejections((long) rejections)
                .falsePositives((long) falsePositives)
                .rejectionRate(rejectionRate)
                .falsePositiveRate(actualFalsePositiveRate)
                .approximateSize(approximateSize)
                .configuredFalsePositiveRate(configuredFalsePositiveRate)
                .build();
    }

    @Data
    @Builder
    public static class BloomStats {
        private long totalChecks;
        private long rejections;
        private long falsePositives;
        private double rejectionRate;
        private double falsePositiveRate;
        private long approximateSize;
        private double configuredFalsePositiveRate;

        public String getRejectionRatePercentage() {
            return String.format("%.2f%%", rejectionRate * 100);
        }

        public String getFalsePositiveRatePercentage() {
            return String.format("%.4f%%", falsePositiveRate * 100);
        }

        public String getConfiguredFalsePositiveRatePercentage() {
            return String.format("%.2f%%", configuredFalsePositiveRate * 100);
        }
    }
}
