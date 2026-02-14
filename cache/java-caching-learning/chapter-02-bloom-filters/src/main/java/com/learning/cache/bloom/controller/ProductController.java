package com.learning.cache.bloom.controller;

import com.learning.cache.bloom.service.BloomProtectedService;
import com.learning.cache.common.dto.ProductDTO;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/products")
@RequiredArgsConstructor
@Slf4j
public class ProductController {

    private final BloomProtectedService bloomProtectedService;

    @GetMapping("/{sku}")
    public ResponseEntity<ProductDTO> getProduct(@PathVariable String sku) {
        log.info("GET /api/products/{} (with Bloom protection)", sku);
        return ResponseEntity.ok(bloomProtectedService.getProduct(sku));
    }

    @GetMapping("/unprotected/{sku}")
    public ResponseEntity<ProductDTO> getProductUnprotected(@PathVariable String sku) {
        log.info("GET /api/products/unprotected/{} (without Bloom protection)", sku);
        return ResponseEntity.ok(bloomProtectedService.getProductUnprotected(sku));
    }

    @PostMapping
    public ResponseEntity<ProductDTO> createProduct(@RequestBody ProductDTO dto) {
        log.info("POST /api/products - creating {}", dto.getSku());
        return ResponseEntity.ok(bloomProtectedService.saveProduct(dto));
    }

    @PutMapping("/{sku}")
    public ResponseEntity<ProductDTO> updateProduct(
            @PathVariable String sku,
            @RequestBody ProductDTO dto) {
        log.info("PUT /api/products/{}", sku);
        dto.setSku(sku);
        return ResponseEntity.ok(bloomProtectedService.saveProduct(dto));
    }

    @DeleteMapping("/{sku}")
    public ResponseEntity<Void> deleteProduct(@PathVariable String sku) {
        log.info("DELETE /api/products/{}", sku);
        bloomProtectedService.deleteProduct(sku);
        return ResponseEntity.noContent().build();
    }
}
