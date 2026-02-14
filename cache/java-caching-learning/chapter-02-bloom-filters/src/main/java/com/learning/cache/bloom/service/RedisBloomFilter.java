package com.learning.cache.bloom.service;

import lombok.extern.slf4j.Slf4j;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.core.RedisCallback;
import org.springframework.data.redis.core.RedisTemplate;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Collection;
import java.util.List;

@Slf4j
public class RedisBloomFilter implements BloomFilterService {

    private final RedisTemplate<String, Object> redisTemplate;
    private final String filterKey;
    private final int expectedInsertions;
    private final double falsePositiveRate;

    public RedisBloomFilter(RedisTemplate<String, Object> redisTemplate,
                           String filterKey,
                           int expectedInsertions,
                           double falsePositiveRate) {
        this.redisTemplate = redisTemplate;
        this.filterKey = filterKey;
        this.expectedInsertions = expectedInsertions;
        this.falsePositiveRate = falsePositiveRate;
        initializeFilter();
    }

    private void initializeFilter() {
        try {
            Boolean exists = redisTemplate.hasKey(filterKey);
            if (exists == null || !exists) {
                createFilter(filterKey);
                log.info("Created new RedisBloom filter: {}", filterKey);
            } else {
                log.info("Using existing RedisBloom filter: {}", filterKey);
            }
        } catch (Exception e) {
            log.error("Failed to initialize RedisBloom filter: {}", e.getMessage());
        }
    }

    @Override
    public boolean mightContain(String item) {
        try {
            Object result = redisTemplate.execute((RedisCallback<Object>) connection ->
                    connection.execute("BF.EXISTS",
                            filterKey.getBytes(StandardCharsets.UTF_8),
                            item.getBytes(StandardCharsets.UTF_8)));

            return result != null && ((Long) result) == 1;
        } catch (Exception e) {
            log.warn("RedisBloom check failed, allowing request: {}", e.getMessage());
            return true; // Fail open
        }
    }

    @Override
    public void add(String item) {
        try {
            redisTemplate.execute((RedisCallback<Object>) connection ->
                    connection.execute("BF.ADD",
                            filterKey.getBytes(StandardCharsets.UTF_8),
                            item.getBytes(StandardCharsets.UTF_8)));
            log.debug("Added item to RedisBloom filter: {}", item);
        } catch (Exception e) {
            log.error("Failed to add item to RedisBloom filter: {}", e.getMessage());
        }
    }

    @Override
    public void addAll(Collection<String> items) {
        if (items.isEmpty()) return;

        try {
            // Use BF.MADD for batch addition
            List<byte[]> args = new ArrayList<>();
            args.add(filterKey.getBytes(StandardCharsets.UTF_8));
            for (String item : items) {
                args.add(item.getBytes(StandardCharsets.UTF_8));
            }

            redisTemplate.execute((RedisCallback<Object>) connection ->
                    connection.execute("BF.MADD", args.toArray(new byte[0][])));

            log.info("Added {} items to RedisBloom filter", items.size());
        } catch (Exception e) {
            log.error("Failed to batch add items to RedisBloom filter: {}", e.getMessage());
            // Fallback to individual adds
            items.forEach(this::add);
        }
    }

    @Override
    public void rebuild(Collection<String> items) {
        String tempKey = filterKey + ":temp:" + System.currentTimeMillis();
        log.info("Rebuilding RedisBloom filter with {} items", items.size());

        try {
            // Create new temporary filter
            createFilter(tempKey);

            // Add all items in batches
            List<String> itemList = new ArrayList<>(items);
            int batchSize = 1000;

            for (int i = 0; i < itemList.size(); i += batchSize) {
                List<String> batch = itemList.subList(i, Math.min(i + batchSize, itemList.size()));
                addAllToFilter(tempKey, batch);
            }

            // Atomic rename
            redisTemplate.rename(tempKey, filterKey);
            log.info("RedisBloom filter rebuild complete");

        } catch (Exception e) {
            log.error("Failed to rebuild RedisBloom filter: {}", e.getMessage());
            redisTemplate.delete(tempKey);
            throw new RuntimeException("Failed to rebuild Bloom filter", e);
        }
    }

    @Override
    public void clear() {
        try {
            redisTemplate.delete(filterKey);
            createFilter(filterKey);
            log.info("RedisBloom filter cleared and recreated");
        } catch (Exception e) {
            log.error("Failed to clear RedisBloom filter: {}", e.getMessage());
        }
    }

    @Override
    public long approximateElementCount() {
        try {
            Object result = redisTemplate.execute((RedisCallback<Object>) connection ->
                    connection.execute("BF.INFO", filterKey.getBytes(StandardCharsets.UTF_8)));

            if (result instanceof List<?> list) {
                // BF.INFO returns alternating key-value pairs
                for (int i = 0; i < list.size() - 1; i += 2) {
                    String key = new String((byte[]) list.get(i), StandardCharsets.UTF_8);
                    if ("Number of items inserted".equals(key)) {
                        return (Long) list.get(i + 1);
                    }
                }
            }
            return 0;
        } catch (Exception e) {
            log.warn("Failed to get element count: {}", e.getMessage());
            return 0;
        }
    }

    @Override
    public double expectedFalsePositiveRate() {
        return falsePositiveRate;
    }

    private void createFilter(String key) {
        redisTemplate.execute((RedisCallback<Object>) connection ->
                connection.execute("BF.RESERVE",
                        key.getBytes(StandardCharsets.UTF_8),
                        String.valueOf(falsePositiveRate).getBytes(StandardCharsets.UTF_8),
                        String.valueOf(expectedInsertions).getBytes(StandardCharsets.UTF_8)));
    }

    private void addAllToFilter(String key, List<String> items) {
        List<byte[]> args = new ArrayList<>();
        args.add(key.getBytes(StandardCharsets.UTF_8));
        for (String item : items) {
            args.add(item.getBytes(StandardCharsets.UTF_8));
        }

        redisTemplate.execute((RedisCallback<Object>) connection ->
                connection.execute("BF.MADD", args.toArray(new byte[0][])));
    }
}
