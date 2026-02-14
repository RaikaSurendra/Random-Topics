package com.learning.cache.consistency.messaging;

import com.learning.cache.consistency.config.RabbitMQConfig;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.UUID;

@Component
@RequiredArgsConstructor
@Slf4j
public class CacheInvalidationPublisher {

    private final RabbitTemplate rabbitTemplate;

    @Value("${spring.application.name:unknown}")
    private String applicationName;

    private final String instanceId = UUID.randomUUID().toString().substring(0, 8);

    public void publishInvalidation(String cacheName, String key) {
        CacheInvalidationEvent event = CacheInvalidationEvent.invalidate(
                cacheName, key, getInstanceId());

        publish(event);
        log.info("Published invalidation event for cache={}, key={}", cacheName, key);
    }

    public void publishUpdate(String cacheName, String key, Object value) {
        CacheInvalidationEvent event = CacheInvalidationEvent.update(
                cacheName, key, value, getInstanceId());

        publish(event);
        log.info("Published update event for cache={}, key={}", cacheName, key);
    }

    public void publishDelete(String cacheName, String key) {
        CacheInvalidationEvent event = CacheInvalidationEvent.delete(
                cacheName, key, getInstanceId());

        publish(event);
        log.info("Published delete event for cache={}, key={}", cacheName, key);
    }

    private void publish(CacheInvalidationEvent event) {
        try {
            rabbitTemplate.convertAndSend(
                    RabbitMQConfig.CACHE_INVALIDATION_EXCHANGE,
                    "",
                    event);
        } catch (Exception e) {
            log.error("Failed to publish cache invalidation event: {}", e.getMessage());
        }
    }

    public String getInstanceId() {
        return applicationName + "-" + instanceId;
    }
}
