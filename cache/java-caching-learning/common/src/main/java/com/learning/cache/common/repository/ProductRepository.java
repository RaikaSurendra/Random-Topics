package com.learning.cache.common.repository;

import com.learning.cache.common.model.Product;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.stereotype.Repository;

import java.util.List;
import java.util.Optional;

@Repository
public interface ProductRepository extends JpaRepository<Product, Long> {

    Optional<Product> findBySku(String sku);

    List<Product> findByCategory(String category);

    @Query("SELECT p.sku FROM Product p")
    List<String> findAllSkus();

    boolean existsBySku(String sku);
}
