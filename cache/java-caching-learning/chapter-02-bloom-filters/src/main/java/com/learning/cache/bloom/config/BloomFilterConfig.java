package com.learning.cache.bloom.config;

import com.learning.cache.bloom.service.BloomFilterService;
import com.learning.cache.bloom.service.InMemoryBloomFilter;
import com.learning.cache.bloom.service.RedisBloomFilter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.core.RedisTemplate;

@Configuration
@Slf4j
public class BloomFilterConfig {

    @Value("${bloom.expected-insertions:100000}")
    private int expectedInsertions;

    @Value("${bloom.false-positive-rate:0.01}")
    private double falsePositiveRate;

    @Value("${bloom.redis.key:products-bloom-filter}")
    private String redisFilterKey;

    @Bean
    @ConditionalOnProperty(name = "bloom.type", havingValue = "memory", matchIfMissing = true)
    public BloomFilterService inMemoryBloomFilter() {
        log.info("Creating in-memory Bloom filter");
        return new InMemoryBloomFilter(expectedInsertions, falsePositiveRate);
    }

    @Bean
    @ConditionalOnProperty(name = "bloom.type", havingValue = "redis")
    public BloomFilterService redisBloomFilter(RedisTemplate<String, Object> redisTemplate) {
        log.info("Creating Redis-based Bloom filter");
        return new RedisBloomFilter(redisTemplate, redisFilterKey, expectedInsertions, falsePositiveRate);
    }
}
