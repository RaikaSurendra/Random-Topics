package com.learning.cache.advanced.service;

import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;
import java.util.stream.Collectors;

@Service
@Slf4j
public class HotKeyDetector {

    @Value("${hotkey.threshold:100}")
    private int hotKeyThreshold;

    @Value("${hotkey.window-seconds:60}")
    private int windowSeconds;

    // Sliding window counters
    private final Cache<String, AtomicLong> accessCounts;

    // Detected hot keys
    private final Set<String> hotKeys = ConcurrentHashMap.newKeySet();

    // Callbacks for hot key detection
    private final List<HotKeyCallback> callbacks = new ArrayList<>();

    public HotKeyDetector() {
        this.accessCounts = Caffeine.newBuilder()
                .expireAfterWrite(Duration.ofSeconds(60))
                .maximumSize(100_000)
                .build();
    }

    public void recordAccess(String key) {
        AtomicLong counter = accessCounts.get(key, k -> new AtomicLong(0));
        long count = counter.incrementAndGet();

        if (count >= hotKeyThreshold && hotKeys.add(key)) {
            log.warn("Hot key detected: {} (access count: {})", key, count);
            notifyCallbacks(key, count);
        }
    }

    public boolean isHotKey(String key) {
        return hotKeys.contains(key);
    }

    public Set<String> getHotKeys() {
        return Collections.unmodifiableSet(hotKeys);
    }

    public List<KeyAccessStats> getTopKeys(int limit) {
        Map<String, Long> counts = new HashMap<>();
        accessCounts.asMap().forEach((key, counter) ->
                counts.put(key, counter.get()));

        return counts.entrySet().stream()
                .sorted(Map.Entry.<String, Long>comparingByValue().reversed())
                .limit(limit)
                .map(e -> new KeyAccessStats(e.getKey(), e.getValue(), hotKeys.contains(e.getKey())))
                .collect(Collectors.toList());
    }

    @Scheduled(fixedRate = 60000) // Every minute
    public void cleanupCoolKeys() {
        // Remove keys that are no longer hot
        hotKeys.removeIf(key -> {
            AtomicLong counter = accessCounts.getIfPresent(key);
            if (counter == null || counter.get() < hotKeyThreshold / 2) {
                log.info("Key {} cooled down, removing from hot keys", key);
                return true;
            }
            return false;
        });
    }

    public void registerCallback(HotKeyCallback callback) {
        callbacks.add(callback);
    }

    private void notifyCallbacks(String key, long count) {
        for (HotKeyCallback callback : callbacks) {
            try {
                callback.onHotKeyDetected(key, count);
            } catch (Exception e) {
                log.error("Hot key callback failed: {}", e.getMessage());
            }
        }
    }

    public record KeyAccessStats(String key, long accessCount, boolean isHot) {}

    @FunctionalInterface
    public interface HotKeyCallback {
        void onHotKeyDetected(String key, long accessCount);
    }
}
