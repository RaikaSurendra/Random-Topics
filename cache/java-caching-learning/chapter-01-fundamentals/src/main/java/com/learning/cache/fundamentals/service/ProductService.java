package com.learning.cache.fundamentals.service;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import com.learning.cache.common.util.SimulatedDelay;
import com.learning.cache.fundamentals.exception.ProductNotFoundException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class ProductService {

    private final ProductRepository productRepository;

    @Transactional(readOnly = true)
    public ProductDTO getProduct(String sku) {
        log.info("Fetching product from database: {}", sku);
        long start = System.currentTimeMillis();

        // Simulate realistic database latency
        SimulatedDelay.databaseQuery();

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new ProductNotFoundException(sku));

        long duration = System.currentTimeMillis() - start;
        log.info("Database query completed in {} ms for SKU: {}", duration, sku);

        return ProductDTO.from(product);
    }

    @Transactional(readOnly = true)
    public List<ProductDTO> getAllProducts() {
        log.info("Fetching all products from database");
        SimulatedDelay.databaseQuery();

        return productRepository.findAll().stream()
                .map(ProductDTO::from)
                .toList();
    }

    @Transactional
    public ProductDTO saveProduct(ProductDTO dto) {
        log.info("Saving product to database: {}", dto.getSku());

        Product product = productRepository.findBySku(dto.getSku())
                .map(existing -> updateExisting(existing, dto))
                .orElseGet(dto::toEntity);

        Product saved = productRepository.save(product);
        log.info("Product saved: {}", saved.getSku());

        return ProductDTO.from(saved);
    }

    @Transactional
    public void deleteProduct(String sku) {
        log.info("Deleting product from database: {}", sku);

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new ProductNotFoundException(sku));

        productRepository.delete(product);
        log.info("Product deleted: {}", sku);
    }

    public boolean existsBySku(String sku) {
        return productRepository.existsBySku(sku);
    }

    public List<String> getAllSkus() {
        return productRepository.findAllSkus();
    }

    private Product updateExisting(Product existing, ProductDTO dto) {
        existing.setName(dto.getName());
        existing.setDescription(dto.getDescription());
        existing.setPrice(dto.getPrice());
        existing.setCategory(dto.getCategory());
        existing.setStockQuantity(dto.getStockQuantity());
        return existing;
    }
}
