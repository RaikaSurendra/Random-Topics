package com.learning.cache.fundamentals.service;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.metrics.CacheMetrics;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import com.learning.cache.common.util.SimulatedDelay;
import com.learning.cache.fundamentals.exception.ProductNotFoundException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.cache.annotation.CacheEvict;
import org.springframework.cache.annotation.CachePut;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

@Service
@RequiredArgsConstructor
@Slf4j
public class CachedProductService {

    private static final String CACHE_NAME = "products";

    private final ProductRepository productRepository;
    private final CacheMetrics cacheMetrics;

    @Cacheable(value = CACHE_NAME, key = "#sku", unless = "#result == null")
    @Transactional(readOnly = true)
    public ProductDTO getProduct(String sku) {
        // This method body only executes on cache MISS
        log.info("Cache MISS for SKU: {} - loading from database", sku);
        cacheMetrics.recordMiss(CACHE_NAME);

        return loadFromDatabase(sku);
    }

    @CachePut(value = CACHE_NAME, key = "#dto.sku")
    @Transactional
    public ProductDTO updateProduct(ProductDTO dto) {
        log.info("Updating product and cache: {}", dto.getSku());

        Product product = productRepository.findBySku(dto.getSku())
                .orElseThrow(() -> new ProductNotFoundException(dto.getSku()));

        product.setName(dto.getName());
        product.setDescription(dto.getDescription());
        product.setPrice(dto.getPrice());
        product.setCategory(dto.getCategory());
        product.setStockQuantity(dto.getStockQuantity());

        Product saved = productRepository.save(product);
        log.info("Product updated in database and cache: {}", saved.getSku());

        return ProductDTO.from(saved);
    }

    @CacheEvict(value = CACHE_NAME, key = "#sku")
    public void evictProduct(String sku) {
        log.info("Evicting product from cache: {}", sku);
        cacheMetrics.recordEviction(CACHE_NAME);
    }

    @CacheEvict(value = CACHE_NAME, allEntries = true)
    public void clearCache() {
        log.info("Clearing entire products cache");
    }

    @Transactional
    @CacheEvict(value = CACHE_NAME, key = "#sku")
    public void deleteProduct(String sku) {
        log.info("Deleting product: {}", sku);

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new ProductNotFoundException(sku));

        productRepository.delete(product);
        cacheMetrics.recordEviction(CACHE_NAME);
        log.info("Product deleted from database and cache: {}", sku);
    }

    public void recordCacheHit(String sku) {
        log.debug("Cache HIT for SKU: {}", sku);
        cacheMetrics.recordHit(CACHE_NAME);
    }

    public CacheMetrics.CacheStats getStats() {
        return cacheMetrics.getStats(CACHE_NAME);
    }

    private ProductDTO loadFromDatabase(String sku) {
        long start = System.nanoTime();

        // Simulate realistic database latency
        SimulatedDelay.databaseQuery();

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new ProductNotFoundException(sku));

        long durationNanos = System.nanoTime() - start;
        cacheMetrics.recordLatency(CACHE_NAME, "database_load", durationNanos);

        log.info("Loaded product from database in {} ms: {}",
                durationNanos / 1_000_000, sku);

        return ProductDTO.from(product);
    }
}
