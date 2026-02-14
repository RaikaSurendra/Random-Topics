package com.learning.cache.distributed.service;

import lombok.extern.slf4j.Slf4j;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.*;
import java.util.concurrent.ConcurrentSkipListMap;

@Slf4j
public class ConsistentHashRing<T> {

    private final int virtualNodes;
    private final ConcurrentSkipListMap<Long, T> ring = new ConcurrentSkipListMap<>();
    private final Map<T, List<Long>> nodePositions = new HashMap<>();

    public ConsistentHashRing(int virtualNodes) {
        this.virtualNodes = virtualNodes;
        log.info("Created consistent hash ring with {} virtual nodes per physical node", virtualNodes);
    }

    public void addNode(T node) {
        List<Long> positions = new ArrayList<>();
        for (int i = 0; i < virtualNodes; i++) {
            long hash = hash(node.toString() + "#" + i);
            ring.put(hash, node);
            positions.add(hash);
        }
        nodePositions.put(node, positions);
        log.info("Added node {} with {} positions on ring", node, virtualNodes);
    }

    public void removeNode(T node) {
        List<Long> positions = nodePositions.remove(node);
        if (positions != null) {
            for (Long position : positions) {
                ring.remove(position);
            }
            log.info("Removed node {} from ring", node);
        }
    }

    public T getNode(String key) {
        if (ring.isEmpty()) {
            return null;
        }

        long hash = hash(key);

        // Find the first node position >= hash
        Map.Entry<Long, T> entry = ring.ceilingEntry(hash);

        // If no entry found, wrap around to the first node
        if (entry == null) {
            entry = ring.firstEntry();
        }

        T node = entry.getValue();
        log.debug("Key '{}' (hash={}) mapped to node {}", key, hash, node);
        return node;
    }

    public List<T> getNodes(String key, int count) {
        if (ring.isEmpty()) {
            return Collections.emptyList();
        }

        Set<T> uniqueNodes = new LinkedHashSet<>();
        long hash = hash(key);

        // Start from the key's position and walk around the ring
        SortedMap<Long, T> tailMap = ring.tailMap(hash);
        for (T node : tailMap.values()) {
            uniqueNodes.add(node);
            if (uniqueNodes.size() >= count) break;
        }

        // Wrap around if needed
        if (uniqueNodes.size() < count) {
            for (T node : ring.values()) {
                uniqueNodes.add(node);
                if (uniqueNodes.size() >= count) break;
            }
        }

        return new ArrayList<>(uniqueNodes);
    }

    public Map<String, Object> getDistribution(List<String> keys) {
        Map<T, Integer> distribution = new HashMap<>();

        for (String key : keys) {
            T node = getNode(key);
            if (node != null) {
                distribution.merge(node, 1, Integer::sum);
            }
        }

        Map<String, Object> result = new HashMap<>();
        result.put("totalKeys", keys.size());
        result.put("nodes", distribution.size());
        result.put("distribution", distribution);

        // Calculate standard deviation
        if (!distribution.isEmpty()) {
            double mean = keys.size() / (double) distribution.size();
            double variance = distribution.values().stream()
                    .mapToDouble(count -> Math.pow(count - mean, 2))
                    .average()
                    .orElse(0);
            result.put("stdDev", Math.sqrt(variance));
            result.put("mean", mean);
        }

        return result;
    }

    public int getNodeCount() {
        return nodePositions.size();
    }

    public Set<T> getAllNodes() {
        return new HashSet<>(nodePositions.keySet());
    }

    public Map<Long, T> getRingSnapshot() {
        return new TreeMap<>(ring);
    }

    private long hash(String key) {
        try {
            MessageDigest md = MessageDigest.getInstance("MD5");
            byte[] digest = md.digest(key.getBytes(StandardCharsets.UTF_8));

            // Use first 8 bytes for a long hash
            long hash = 0;
            for (int i = 0; i < 8; i++) {
                hash = (hash << 8) | (digest[i] & 0xFF);
            }
            return hash;
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("MD5 not available", e);
        }
    }
}
