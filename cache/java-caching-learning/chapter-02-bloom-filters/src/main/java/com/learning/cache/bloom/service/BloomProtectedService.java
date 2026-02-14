package com.learning.cache.bloom.service;

import com.learning.cache.bloom.exception.ProductNotFoundException;
import com.learning.cache.bloom.metrics.BloomMetrics;
import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import com.learning.cache.common.util.SimulatedDelay;
import jakarta.annotation.PostConstruct;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.cache.annotation.CacheEvict;
import org.springframework.cache.annotation.CachePut;
import org.springframework.cache.annotation.Cacheable;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class BloomProtectedService {

    private static final String CACHE_NAME = "products";

    private final BloomFilterService bloomFilter;
    private final ProductRepository productRepository;
    private final BloomMetrics bloomMetrics;

    @PostConstruct
    public void initialize() {
        log.info("Initializing Bloom filter with existing products");
        rebuildFilter();
    }

    public ProductDTO getProduct(String sku) {
        bloomMetrics.recordCheck();

        // Step 1: Bloom filter check
        if (!bloomFilter.mightContain(sku)) {
            bloomMetrics.recordRejection();
            log.debug("Bloom filter rejected SKU: {}", sku);
            throw new ProductNotFoundException(sku);
        }

        // Step 2: Standard cache lookup
        try {
            return getFromCacheOrDb(sku);
        } catch (ProductNotFoundException e) {
            // This was a false positive
            bloomMetrics.recordFalsePositive();
            throw e;
        }
    }

    @Cacheable(value = CACHE_NAME, key = "#sku", unless = "#result == null")
    @Transactional(readOnly = true)
    public ProductDTO getFromCacheOrDb(String sku) {
        log.info("Cache miss for SKU: {} - loading from database", sku);
        SimulatedDelay.databaseQuery();

        return productRepository.findBySku(sku)
                .map(ProductDTO::from)
                .orElseThrow(() -> new ProductNotFoundException(sku));
    }

    @Transactional(readOnly = true)
    public ProductDTO getProductUnprotected(String sku) {
        log.info("Unprotected access for SKU: {}", sku);
        SimulatedDelay.databaseQuery();

        return productRepository.findBySku(sku)
                .map(ProductDTO::from)
                .orElseThrow(() -> new ProductNotFoundException(sku));
    }

    @CachePut(value = CACHE_NAME, key = "#dto.sku")
    @Transactional
    public ProductDTO saveProduct(ProductDTO dto) {
        log.info("Saving product: {}", dto.getSku());

        Product product = productRepository.findBySku(dto.getSku())
                .map(existing -> updateProduct(existing, dto))
                .orElseGet(dto::toEntity);

        Product saved = productRepository.save(product);

        // Add to Bloom filter
        bloomFilter.add(saved.getSku());
        log.info("Added SKU to Bloom filter: {}", saved.getSku());

        return ProductDTO.from(saved);
    }

    @CacheEvict(value = CACHE_NAME, key = "#sku")
    @Transactional
    public void deleteProduct(String sku) {
        log.info("Deleting product: {}", sku);

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new ProductNotFoundException(sku));

        productRepository.delete(product);

        // Note: Cannot remove from Bloom filter (by design)
        // Will be cleaned up on next rebuild
        log.info("Product deleted: {}", sku);
    }

    @Scheduled(cron = "${bloom.rebuild-cron:0 0 3 * * *}") // Default: 3 AM daily
    public void scheduledRebuild() {
        log.info("Scheduled Bloom filter rebuild triggered");
        rebuildFilter();
    }

    public void rebuildFilter() {
        log.info("Rebuilding Bloom filter...");
        List<String> allSkus = productRepository.findAllSkus();
        bloomFilter.rebuild(allSkus);
        log.info("Bloom filter rebuilt with {} SKUs", allSkus.size());
    }

    public BloomMetrics.BloomStats getStats() {
        return bloomMetrics.getStats(
                bloomFilter.approximateElementCount(),
                bloomFilter.expectedFalsePositiveRate()
        );
    }

    private Product updateProduct(Product existing, ProductDTO dto) {
        existing.setName(dto.getName());
        existing.setDescription(dto.getDescription());
        existing.setPrice(dto.getPrice());
        existing.setCategory(dto.getCategory());
        existing.setStockQuantity(dto.getStockQuantity());
        return existing;
    }
}
