package com.learning.cache.consistency.messaging;

import com.learning.cache.consistency.config.RabbitMQConfig;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Component;

import java.util.concurrent.atomic.AtomicLong;

@Component
@RequiredArgsConstructor
@Slf4j
public class CacheInvalidationListener {

    private final RedisTemplate<String, Object> redisTemplate;
    private final CacheInvalidationPublisher publisher;

    private final AtomicLong eventsProcessed = new AtomicLong(0);
    private final AtomicLong eventsIgnored = new AtomicLong(0);

    @RabbitListener(queues = RabbitMQConfig.CACHE_INVALIDATION_QUEUE)
    public void handleInvalidationEvent(CacheInvalidationEvent event) {
        // Skip events from self
        if (event.getSourceInstanceId().equals(publisher.getInstanceId())) {
            eventsIgnored.incrementAndGet();
            log.debug("Ignoring self-published event for key={}", event.getKey());
            return;
        }

        long latency = System.currentTimeMillis() - event.getTimestamp();
        log.info("Received {} event for cache={}, key={} (latency={}ms)",
                event.getType(), event.getCacheName(), event.getKey(), latency);

        try {
            String cacheKey = event.getCacheName() + "::" + event.getKey();

            switch (event.getType()) {
                case INVALIDATE, DELETE -> {
                    redisTemplate.delete(cacheKey);
                    log.debug("Deleted cache key: {}", cacheKey);
                }
                case UPDATE -> {
                    if (event.getNewValue() != null) {
                        redisTemplate.opsForValue().set(cacheKey, event.getNewValue());
                        log.debug("Updated cache key: {}", cacheKey);
                    }
                }
            }

            eventsProcessed.incrementAndGet();
        } catch (Exception e) {
            log.error("Failed to process invalidation event: {}", e.getMessage());
        }
    }

    public long getEventsProcessed() {
        return eventsProcessed.get();
    }

    public long getEventsIgnored() {
        return eventsIgnored.get();
    }
}
