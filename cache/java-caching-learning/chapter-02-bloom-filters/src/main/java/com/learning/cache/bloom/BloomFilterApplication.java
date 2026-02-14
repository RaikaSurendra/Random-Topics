package com.learning.cache.bloom;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.cache.annotation.EnableCaching;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication(scanBasePackages = {
    "com.learning.cache.bloom",
    "com.learning.cache.common"
})
@EnableCaching
@EnableScheduling
@EntityScan(basePackages = "com.learning.cache.common.model")
@EnableJpaRepositories(basePackages = {
    "com.learning.cache.bloom.repository",
    "com.learning.cache.common.repository"
})
public class BloomFilterApplication {

    public static void main(String[] args) {
        SpringApplication.run(BloomFilterApplication.class, args);
    }
}
