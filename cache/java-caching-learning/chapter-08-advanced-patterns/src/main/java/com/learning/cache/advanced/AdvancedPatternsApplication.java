package com.learning.cache.advanced;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication(scanBasePackages = {
    "com.learning.cache.advanced",
    "com.learning.cache.common"
})
@EnableScheduling
@EntityScan(basePackages = "com.learning.cache.common.model")
@EnableJpaRepositories(basePackages = {
    "com.learning.cache.advanced.repository",
    "com.learning.cache.common.repository"
})
public class AdvancedPatternsApplication {

    public static void main(String[] args) {
        SpringApplication.run(AdvancedPatternsApplication.class, args);
    }
}
