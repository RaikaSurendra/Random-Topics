package com.learning.cache.consistency.messaging;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.io.Serializable;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class CacheInvalidationEvent implements Serializable {

    private static final long serialVersionUID = 1L;

    public enum EventType {
        INVALIDATE,
        UPDATE,
        DELETE
    }

    private EventType type;
    private String cacheName;
    private String key;
    private Object newValue;
    private long timestamp;
    private String sourceInstanceId;

    public static CacheInvalidationEvent invalidate(String cacheName, String key, String instanceId) {
        return CacheInvalidationEvent.builder()
                .type(EventType.INVALIDATE)
                .cacheName(cacheName)
                .key(key)
                .timestamp(System.currentTimeMillis())
                .sourceInstanceId(instanceId)
                .build();
    }

    public static CacheInvalidationEvent update(String cacheName, String key, Object value, String instanceId) {
        return CacheInvalidationEvent.builder()
                .type(EventType.UPDATE)
                .cacheName(cacheName)
                .key(key)
                .newValue(value)
                .timestamp(System.currentTimeMillis())
                .sourceInstanceId(instanceId)
                .build();
    }

    public static CacheInvalidationEvent delete(String cacheName, String key, String instanceId) {
        return CacheInvalidationEvent.builder()
                .type(EventType.DELETE)
                .cacheName(cacheName)
                .key(key)
                .timestamp(System.currentTimeMillis())
                .sourceInstanceId(instanceId)
                .build();
    }
}
