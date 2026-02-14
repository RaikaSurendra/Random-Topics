package com.learning.cache.advanced.service;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import jakarta.annotation.PostConstruct;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.List;
import java.util.concurrent.atomic.AtomicBoolean;

@Service
@RequiredArgsConstructor
@Slf4j
public class CacheWarmer {

    private static final String CACHE_PREFIX = "warm:product:";
    private static final Duration TTL = Duration.ofMinutes(10);

    private final ProductRepository productRepository;
    private final RedisTemplate<String, Object> redisTemplate;

    @Value("${cache.warmer.count:100}")
    private int warmCount;

    @Value("${cache.warmer.enabled:true}")
    private boolean enabled;

    private final AtomicBoolean warmed = new AtomicBoolean(false);

    @PostConstruct
    public void warmOnStartup() {
        if (enabled) {
            warmCache();
        }
    }

    @Scheduled(cron = "${cache.warmer.cron:0 0 5 * * *}") // 5 AM daily
    public void scheduledWarm() {
        if (enabled) {
            warmCache();
        }
    }

    public void warmCache() {
        log.info("Starting cache warming...");
        long start = System.currentTimeMillis();

        try {
            List<Product> products = productRepository.findAll();

            // In production, you'd select top N by popularity
            List<Product> toWarm = products.stream()
                    .limit(warmCount)
                    .toList();

            int warmedCount = 0;
            for (Product product : toWarm) {
                try {
                    String key = CACHE_PREFIX + product.getSku();
                    ProductDTO dto = ProductDTO.from(product);
                    redisTemplate.opsForValue().set(key, dto, TTL);
                    warmedCount++;
                } catch (Exception e) {
                    log.warn("Failed to warm cache for SKU {}: {}", product.getSku(), e.getMessage());
                }
            }

            long duration = System.currentTimeMillis() - start;
            warmed.set(true);
            log.info("Cache warming complete: {} products in {} ms", warmedCount, duration);

        } catch (Exception e) {
            log.error("Cache warming failed: {}", e.getMessage());
        }
    }

    public ProductDTO getProduct(String sku) {
        String key = CACHE_PREFIX + sku;
        return (ProductDTO) redisTemplate.opsForValue().get(key);
    }

    public boolean isWarmed() {
        return warmed.get();
    }

    public WarmingStats getStats() {
        // Count warmed keys
        long keyCount = redisTemplate.keys(CACHE_PREFIX + "*").size();
        return new WarmingStats(warmed.get(), keyCount, warmCount);
    }

    public record WarmingStats(boolean warmed, long currentCount, int targetCount) {
        public double getWarmPercentage() {
            return targetCount > 0 ? (double) currentCount / targetCount * 100 : 0;
        }
    }
}
