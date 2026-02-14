package com.learning.cache.fundamentals.controller;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.fundamentals.service.CachedProductService;
import com.learning.cache.fundamentals.service.ProductService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api/products")
@RequiredArgsConstructor
@Slf4j
public class ProductController {

    private final CachedProductService cachedProductService;
    private final ProductService productService;

    @GetMapping("/{sku}")
    public ResponseEntity<ProductDTO> getProduct(@PathVariable String sku) {
        log.info("GET /api/products/{}", sku);
        long start = System.currentTimeMillis();

        ProductDTO product = cachedProductService.getProduct(sku);

        long duration = System.currentTimeMillis() - start;
        log.info("Request completed in {} ms", duration);

        return ResponseEntity.ok(product);
    }

    @GetMapping("/uncached/{sku}")
    public ResponseEntity<ProductDTO> getProductUncached(@PathVariable String sku) {
        log.info("GET /api/products/uncached/{} (bypassing cache)", sku);
        long start = System.currentTimeMillis();

        ProductDTO product = productService.getProduct(sku);

        long duration = System.currentTimeMillis() - start;
        log.info("Uncached request completed in {} ms", duration);

        return ResponseEntity.ok(product);
    }

    @GetMapping
    public ResponseEntity<List<ProductDTO>> getAllProducts() {
        log.info("GET /api/products");
        return ResponseEntity.ok(productService.getAllProducts());
    }

    @PutMapping("/{sku}")
    public ResponseEntity<ProductDTO> updateProduct(
            @PathVariable String sku,
            @RequestBody ProductDTO dto) {
        log.info("PUT /api/products/{}", sku);

        dto.setSku(sku);
        ProductDTO updated = cachedProductService.updateProduct(dto);

        return ResponseEntity.ok(updated);
    }

    @DeleteMapping("/{sku}")
    public ResponseEntity<Void> deleteProduct(@PathVariable String sku) {
        log.info("DELETE /api/products/{}", sku);

        cachedProductService.deleteProduct(sku);

        return ResponseEntity.noContent().build();
    }

    @DeleteMapping("/cache/{sku}")
    public ResponseEntity<Void> evictCache(@PathVariable String sku) {
        log.info("DELETE /api/products/cache/{} (evicting cache only)", sku);

        cachedProductService.evictProduct(sku);

        return ResponseEntity.noContent().build();
    }

    @DeleteMapping("/cache")
    public ResponseEntity<Void> clearCache() {
        log.info("DELETE /api/products/cache (clearing all cache)");

        cachedProductService.clearCache();

        return ResponseEntity.noContent().build();
    }
}
