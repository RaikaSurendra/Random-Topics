package com.learning.cache.common.dto;

import com.learning.cache.common.model.Product;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.io.Serializable;
import java.math.BigDecimal;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ProductDTO implements Serializable {

    private static final long serialVersionUID = 1L;

    private Long id;
    private String sku;
    private String name;
    private String description;
    private BigDecimal price;
    private String category;
    private Integer stockQuantity;

    public static ProductDTO from(Product product) {
        if (product == null) {
            return null;
        }
        return ProductDTO.builder()
                .id(product.getId())
                .sku(product.getSku())
                .name(product.getName())
                .description(product.getDescription())
                .price(product.getPrice())
                .category(product.getCategory())
                .stockQuantity(product.getStockQuantity())
                .build();
    }

    public Product toEntity() {
        return Product.builder()
                .id(id)
                .sku(sku)
                .name(name)
                .description(description)
                .price(price)
                .category(category)
                .stockQuantity(stockQuantity)
                .build();
    }
}
