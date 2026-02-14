package com.learning.cache.distributed.service;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import com.learning.cache.common.util.SimulatedDelay;
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.circuitbreaker.CircuitBreakerConfig;
import io.github.resilience4j.circuitbreaker.CircuitBreakerRegistry;
import jakarta.annotation.PostConstruct;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.data.redis.connection.lettuce.LettuceConnectionFactory;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.serializer.GenericJackson2JsonRedisSerializer;
import org.springframework.data.redis.serializer.StringRedisSerializer;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

@Service
@RequiredArgsConstructor
@Slf4j
public class ShardedCacheService {

    private static final String CACHE_PREFIX = "shard:product:";
    private static final Duration TTL = Duration.ofSeconds(60);

    private final ProductRepository productRepository;
    private final CircuitBreakerRegistry circuitBreakerRegistry;

    @Value("${distributed.nodes:localhost:6379,localhost:6380,localhost:6381}")
    private String nodeConfig;

    @Value("${distributed.virtual-nodes:150}")
    private int virtualNodes;

    private ConsistentHashRing<String> hashRing;
    private final Map<String, RedisTemplate<String, Object>> nodeTemplates = new ConcurrentHashMap<>();
    private final Map<String, CircuitBreaker> circuitBreakers = new ConcurrentHashMap<>();

    @PostConstruct
    public void initialize() {
        hashRing = new ConsistentHashRing<>(virtualNodes);

        String[] nodes = nodeConfig.split(",");
        for (String node : nodes) {
            addNode(node.trim());
        }

        log.info("Initialized sharded cache with {} nodes", nodes.length);
    }

    public void addNode(String nodeAddress) {
        hashRing.addNode(nodeAddress);
        nodeTemplates.put(nodeAddress, createRedisTemplate(nodeAddress));
        circuitBreakers.put(nodeAddress, createCircuitBreaker(nodeAddress));
        log.info("Added cache node: {}", nodeAddress);
    }

    public void removeNode(String nodeAddress) {
        hashRing.removeNode(nodeAddress);
        nodeTemplates.remove(nodeAddress);
        circuitBreakers.remove(nodeAddress);
        log.info("Removed cache node: {}", nodeAddress);
    }

    public ProductDTO getProduct(String sku) {
        String nodeAddress = hashRing.getNode(sku);
        if (nodeAddress == null) {
            log.warn("No nodes available, falling back to database");
            return loadFromDatabase(sku);
        }

        CircuitBreaker circuitBreaker = circuitBreakers.get(nodeAddress);
        if (circuitBreaker == null) {
            return loadFromDatabase(sku);
        }

        try {
            return circuitBreaker.executeSupplier(() -> getFromNode(nodeAddress, sku));
        } catch (Exception e) {
            log.warn("Node {} failed, trying fallback: {}", nodeAddress, e.getMessage());
            return fallbackGet(sku, nodeAddress);
        }
    }

    private ProductDTO getFromNode(String nodeAddress, String sku) {
        RedisTemplate<String, Object> template = nodeTemplates.get(nodeAddress);
        String cacheKey = CACHE_PREFIX + sku;

        ProductDTO cached = (ProductDTO) template.opsForValue().get(cacheKey);
        if (cached != null) {
            log.debug("Cache HIT on node {} for {}", nodeAddress, sku);
            return cached;
        }

        log.debug("Cache MISS on node {} for {}", nodeAddress, sku);
        ProductDTO product = loadFromDatabase(sku);

        template.opsForValue().set(cacheKey, product, TTL);
        return product;
    }

    private ProductDTO fallbackGet(String sku, String failedNode) {
        // Try next nodes in the ring
        List<String> nodes = hashRing.getNodes(sku, 3);
        for (String node : nodes) {
            if (node.equals(failedNode)) continue;

            try {
                CircuitBreaker cb = circuitBreakers.get(node);
                if (cb != null && cb.getState() != CircuitBreaker.State.OPEN) {
                    return cb.executeSupplier(() -> getFromNode(node, sku));
                }
            } catch (Exception e) {
                log.warn("Fallback node {} also failed: {}", node, e.getMessage());
            }
        }

        // All nodes failed, load from database
        log.warn("All cache nodes failed, loading from database");
        return loadFromDatabase(sku);
    }

    public void putProduct(String sku, ProductDTO product) {
        String nodeAddress = hashRing.getNode(sku);
        if (nodeAddress != null) {
            RedisTemplate<String, Object> template = nodeTemplates.get(nodeAddress);
            if (template != null) {
                template.opsForValue().set(CACHE_PREFIX + sku, product, TTL);
            }
        }
    }

    public void evictProduct(String sku) {
        String nodeAddress = hashRing.getNode(sku);
        if (nodeAddress != null) {
            RedisTemplate<String, Object> template = nodeTemplates.get(nodeAddress);
            if (template != null) {
                template.delete(CACHE_PREFIX + sku);
            }
        }
    }

    public String getNodeForKey(String key) {
        return hashRing.getNode(key);
    }

    public Map<String, Object> getDistribution(List<String> keys) {
        return hashRing.getDistribution(keys);
    }

    public Map<String, String> getNodeStates() {
        Map<String, String> states = new HashMap<>();
        for (Map.Entry<String, CircuitBreaker> entry : circuitBreakers.entrySet()) {
            states.put(entry.getKey(), entry.getValue().getState().name());
        }
        return states;
    }

    private ProductDTO loadFromDatabase(String sku) {
        SimulatedDelay.databaseQuery();
        return productRepository.findBySku(sku)
                .map(ProductDTO::from)
                .orElseThrow(() -> new RuntimeException("Product not found: " + sku));
    }

    private RedisTemplate<String, Object> createRedisTemplate(String nodeAddress) {
        String[] parts = nodeAddress.split(":");
        String host = parts[0];
        int port = parts.length > 1 ? Integer.parseInt(parts[1]) : 6379;

        LettuceConnectionFactory factory = new LettuceConnectionFactory(host, port);
        factory.afterPropertiesSet();

        RedisTemplate<String, Object> template = new RedisTemplate<>();
        template.setConnectionFactory(factory);
        template.setKeySerializer(new StringRedisSerializer());
        template.setValueSerializer(new GenericJackson2JsonRedisSerializer());
        template.afterPropertiesSet();

        return template;
    }

    private CircuitBreaker createCircuitBreaker(String nodeAddress) {
        CircuitBreakerConfig config = CircuitBreakerConfig.custom()
                .failureRateThreshold(50)
                .waitDurationInOpenState(Duration.ofSeconds(30))
                .slidingWindowSize(10)
                .minimumNumberOfCalls(5)
                .build();

        return circuitBreakerRegistry.circuitBreaker(nodeAddress, config);
    }
}
