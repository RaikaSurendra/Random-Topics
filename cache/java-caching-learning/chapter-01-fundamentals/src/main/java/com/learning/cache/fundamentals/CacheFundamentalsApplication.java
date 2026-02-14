package com.learning.cache.fundamentals;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.cache.annotation.EnableCaching;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;

@SpringBootApplication(scanBasePackages = {
    "com.learning.cache.fundamentals",
    "com.learning.cache.common"
})
@EnableCaching
@EntityScan(basePackages = "com.learning.cache.common.model")
@EnableJpaRepositories(basePackages = {
    "com.learning.cache.fundamentals.repository",
    "com.learning.cache.common.repository"
})
public class CacheFundamentalsApplication {

    public static void main(String[] args) {
        SpringApplication.run(CacheFundamentalsApplication.class, args);
    }
}
