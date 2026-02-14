package com.learning.cache.advanced.service;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import com.learning.cache.common.util.SimulatedDelay;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.Set;
import java.util.concurrent.atomic.AtomicReference;

@Service
@RequiredArgsConstructor
@Slf4j
public class VersionedCacheService {

    private static final Duration TTL = Duration.ofMinutes(10);

    private final RedisTemplate<String, Object> redisTemplate;
    private final ProductRepository productRepository;

    @Value("${cache.version:v1}")
    private String initialVersion;

    private final AtomicReference<String> currentVersion = new AtomicReference<>();

    public void initializeVersion() {
        if (currentVersion.get() == null) {
            currentVersion.set(initialVersion);
        }
    }

    public ProductDTO getProduct(String sku) {
        initializeVersion();
        String key = buildKey(sku);

        ProductDTO cached = (ProductDTO) redisTemplate.opsForValue().get(key);
        if (cached != null) {
            log.debug("Cache HIT for {} (version: {})", sku, currentVersion.get());
            return cached;
        }

        log.debug("Cache MISS for {} (version: {})", sku, currentVersion.get());
        SimulatedDelay.databaseQuery();

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new RuntimeException("Product not found: " + sku));

        ProductDTO dto = ProductDTO.from(product);
        redisTemplate.opsForValue().set(key, dto, TTL);

        return dto;
    }

    public void switchVersion(String newVersion) {
        String oldVersion = currentVersion.getAndSet(newVersion);
        log.info("Switched cache version: {} -> {}", oldVersion, newVersion);

        // Optionally clean up old version keys in background
        cleanupOldVersion(oldVersion);
    }

    public String getCurrentVersion() {
        initializeVersion();
        return currentVersion.get();
    }

    public VersionStats getVersionStats() {
        initializeVersion();
        String version = currentVersion.get();
        String pattern = version + "::products::*";

        Set<String> keys = redisTemplate.keys(pattern);
        long keyCount = keys != null ? keys.size() : 0;

        return new VersionStats(version, keyCount);
    }

    private String buildKey(String sku) {
        return currentVersion.get() + "::products::" + sku;
    }

    private void cleanupOldVersion(String oldVersion) {
        if (oldVersion == null) return;

        // Run in background
        new Thread(() -> {
            try {
                String pattern = oldVersion + "::products::*";
                Set<String> oldKeys = redisTemplate.keys(pattern);
                if (oldKeys != null && !oldKeys.isEmpty()) {
                    redisTemplate.delete(oldKeys);
                    log.info("Cleaned up {} keys from old version: {}", oldKeys.size(), oldVersion);
                }
            } catch (Exception e) {
                log.error("Failed to cleanup old version: {}", e.getMessage());
            }
        }).start();
    }

    public record VersionStats(String currentVersion, long keyCount) {}
}
