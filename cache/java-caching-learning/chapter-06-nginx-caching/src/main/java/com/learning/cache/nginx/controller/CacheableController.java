package com.learning.cache.nginx.controller;

import com.learning.cache.common.dto.ProductDTO;
import com.learning.cache.common.model.Product;
import com.learning.cache.common.repository.ProductRepository;
import com.learning.cache.common.util.SimulatedDelay;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.CacheControl;
import org.springframework.http.HttpHeaders;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.concurrent.TimeUnit;

@RestController
@RequestMapping("/api/products")
@RequiredArgsConstructor
@Slf4j
public class CacheableController {

    private final ProductRepository productRepository;

    @GetMapping("/{sku}")
    public ResponseEntity<ProductDTO> getProduct(@PathVariable String sku) {
        log.info("Fetching product: {} (origin request)", sku);
        SimulatedDelay.databaseQuery();

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new RuntimeException("Product not found: " + sku));

        return ResponseEntity.ok()
                .cacheControl(CacheControl.maxAge(60, TimeUnit.SECONDS).cachePublic())
                .header("X-Origin-Time", Instant.now().toString())
                .body(ProductDTO.from(product));
    }

    @GetMapping("/no-cache/{sku}")
    public ResponseEntity<ProductDTO> getProductNoCache(@PathVariable String sku) {
        log.info("Fetching product (no cache): {}", sku);
        SimulatedDelay.databaseQuery();

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new RuntimeException("Product not found: " + sku));

        return ResponseEntity.ok()
                .cacheControl(CacheControl.noStore())
                .header("X-Origin-Time", Instant.now().toString())
                .body(ProductDTO.from(product));
    }

    @GetMapping("/microcache/{sku}")
    public ResponseEntity<ProductDTO> getProductMicrocache(@PathVariable String sku) {
        log.info("Fetching product (microcache): {}", sku);
        SimulatedDelay.databaseQuery();

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new RuntimeException("Product not found: " + sku));

        // Microcaching: very short TTL (1 second) for semi-dynamic content
        return ResponseEntity.ok()
                .cacheControl(CacheControl.maxAge(1, TimeUnit.SECONDS)
                        .sMaxAge(Duration.ofSeconds(1))
                        .cachePublic())
                .header("X-Origin-Time", Instant.now().toString())
                .body(ProductDTO.from(product));
    }

    @GetMapping
    public ResponseEntity<List<ProductDTO>> getAllProducts() {
        log.info("Fetching all products (origin request)");
        SimulatedDelay.databaseQuery();

        List<ProductDTO> products = productRepository.findAll().stream()
                .map(ProductDTO::from)
                .toList();

        return ResponseEntity.ok()
                .cacheControl(CacheControl.maxAge(30, TimeUnit.SECONDS).cachePublic())
                .body(products);
    }

    @PutMapping("/{sku}")
    public ResponseEntity<ProductDTO> updateProduct(@PathVariable String sku,
                                                    @RequestBody ProductDTO dto) {
        log.info("Updating product: {}", sku);

        Product product = productRepository.findBySku(sku)
                .orElseThrow(() -> new RuntimeException("Product not found: " + sku));

        product.setName(dto.getName());
        product.setPrice(dto.getPrice());
        Product saved = productRepository.save(product);

        return ResponseEntity.ok()
                .cacheControl(CacheControl.noStore())
                .body(ProductDTO.from(saved));
    }
}
