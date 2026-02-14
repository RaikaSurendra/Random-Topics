package com.learning.cache.common.util;

import lombok.Builder;
import lombok.Data;
import lombok.extern.slf4j.Slf4j;

import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.LongSummaryStatistics;
import java.util.concurrent.*;
import java.util.function.Consumer;
import java.util.function.Supplier;
import java.util.stream.IntStream;

@Slf4j
public class LoadGenerator {

    private final ExecutorService executor;

    public LoadGenerator(int threadPoolSize) {
        this.executor = Executors.newFixedThreadPool(threadPoolSize);
    }

    public LoadGenerator() {
        this(Runtime.getRuntime().availableProcessors() * 2);
    }

    public <T> LoadResult<T> runConcurrent(int requestCount, Supplier<T> task) {
        return runConcurrent(requestCount, task, Duration.ofMinutes(5));
    }

    public <T> LoadResult<T> runConcurrent(int requestCount, Supplier<T> task, Duration timeout) {
        List<Future<TimedResult<T>>> futures = new ArrayList<>();
        CountDownLatch startLatch = new CountDownLatch(1);
        CountDownLatch endLatch = new CountDownLatch(requestCount);

        Instant overallStart = Instant.now();

        // Submit all tasks
        for (int i = 0; i < requestCount; i++) {
            futures.add(executor.submit(() -> {
                startLatch.await(); // Wait for all threads to be ready
                long start = System.nanoTime();
                try {
                    T result = task.get();
                    return new TimedResult<>(result, System.nanoTime() - start, null);
                } catch (Exception e) {
                    return new TimedResult<>(null, System.nanoTime() - start, e);
                } finally {
                    endLatch.countDown();
                }
            }));
        }

        // Release all threads simultaneously
        startLatch.countDown();

        // Wait for completion
        try {
            boolean completed = endLatch.await(timeout.toMillis(), TimeUnit.MILLISECONDS);
            if (!completed) {
                log.warn("Load test timed out after {}", timeout);
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            log.error("Load test interrupted", e);
        }

        Duration overallDuration = Duration.between(overallStart, Instant.now());

        // Collect results
        List<TimedResult<T>> results = new ArrayList<>();
        int successCount = 0;
        int errorCount = 0;
        List<Long> latencies = new ArrayList<>();

        for (Future<TimedResult<T>> future : futures) {
            try {
                TimedResult<T> result = future.get(1, TimeUnit.SECONDS);
                results.add(result);
                latencies.add(result.getDurationNanos());
                if (result.getError() == null) {
                    successCount++;
                } else {
                    errorCount++;
                }
            } catch (Exception e) {
                errorCount++;
            }
        }

        // Calculate statistics
        LongSummaryStatistics stats = latencies.stream()
                .mapToLong(Long::longValue)
                .summaryStatistics();

        latencies.sort(Long::compareTo);
        long p50 = latencies.isEmpty() ? 0 : latencies.get((int) (latencies.size() * 0.50));
        long p95 = latencies.isEmpty() ? 0 : latencies.get((int) (latencies.size() * 0.95));
        long p99 = latencies.isEmpty() ? 0 : latencies.get((int) (latencies.size() * 0.99));

        return LoadResult.<T>builder()
                .requestCount(requestCount)
                .successCount(successCount)
                .errorCount(errorCount)
                .totalDuration(overallDuration)
                .avgLatencyNanos((long) stats.getAverage())
                .minLatencyNanos(stats.getMin())
                .maxLatencyNanos(stats.getMax())
                .p50LatencyNanos(p50)
                .p95LatencyNanos(p95)
                .p99LatencyNanos(p99)
                .requestsPerSecond(requestCount / (overallDuration.toMillis() / 1000.0))
                .results(results)
                .build();
    }

    public void runSustainedLoad(int requestsPerSecond, Duration duration, Runnable task) {
        runSustainedLoad(requestsPerSecond, duration, task, result -> {});
    }

    public void runSustainedLoad(int requestsPerSecond, Duration duration, Runnable task,
                                  Consumer<LoadResult<Void>> progressCallback) {
        ScheduledExecutorService scheduler = Executors.newScheduledThreadPool(1);
        long intervalMicros = 1_000_000 / requestsPerSecond;

        Instant endTime = Instant.now().plus(duration);
        int batchSize = Math.max(1, requestsPerSecond / 10); // Report every 100ms worth

        scheduler.scheduleAtFixedRate(() -> {
            if (Instant.now().isAfter(endTime)) {
                scheduler.shutdown();
                return;
            }

            LoadResult<Void> batchResult = runConcurrent(batchSize, () -> {
                task.run();
                return null;
            });

            progressCallback.accept(batchResult);
        }, 0, 100, TimeUnit.MILLISECONDS);

        try {
            scheduler.awaitTermination(duration.toMillis() + 5000, TimeUnit.MILLISECONDS);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }

    public void generateRandomKeyLoad(int keyCount, int requestCount, Consumer<String> keyConsumer) {
        List<String> keys = IntStream.range(0, keyCount)
                .mapToObj(i -> "key-" + i)
                .toList();

        ThreadLocalRandom random = ThreadLocalRandom.current();

        runConcurrent(requestCount, () -> {
            String key = keys.get(random.nextInt(keys.size()));
            keyConsumer.accept(key);
            return key;
        });
    }

    public void shutdown() {
        executor.shutdown();
        try {
            if (!executor.awaitTermination(10, TimeUnit.SECONDS)) {
                executor.shutdownNow();
            }
        } catch (InterruptedException e) {
            executor.shutdownNow();
            Thread.currentThread().interrupt();
        }
    }

    @Data
    @Builder
    public static class LoadResult<T> {
        private int requestCount;
        private int successCount;
        private int errorCount;
        private Duration totalDuration;
        private long avgLatencyNanos;
        private long minLatencyNanos;
        private long maxLatencyNanos;
        private long p50LatencyNanos;
        private long p95LatencyNanos;
        private long p99LatencyNanos;
        private double requestsPerSecond;
        private List<TimedResult<T>> results;

        public String getSummary() {
            return String.format(
                    "Requests: %d (success: %d, errors: %d)\n" +
                    "Duration: %d ms\n" +
                    "Throughput: %.2f req/s\n" +
                    "Latency (ms): avg=%.2f, min=%.2f, max=%.2f\n" +
                    "Percentiles (ms): p50=%.2f, p95=%.2f, p99=%.2f",
                    requestCount, successCount, errorCount,
                    totalDuration.toMillis(),
                    requestsPerSecond,
                    avgLatencyNanos / 1_000_000.0,
                    minLatencyNanos / 1_000_000.0,
                    maxLatencyNanos / 1_000_000.0,
                    p50LatencyNanos / 1_000_000.0,
                    p95LatencyNanos / 1_000_000.0,
                    p99LatencyNanos / 1_000_000.0
            );
        }

        public double getSuccessRate() {
            return requestCount == 0 ? 0 : (double) successCount / requestCount;
        }
    }

    @Data
    public static class TimedResult<T> {
        private final T result;
        private final long durationNanos;
        private final Exception error;

        public boolean isSuccess() {
            return error == null;
        }

        public double getDurationMillis() {
            return durationNanos / 1_000_000.0;
        }
    }
}
