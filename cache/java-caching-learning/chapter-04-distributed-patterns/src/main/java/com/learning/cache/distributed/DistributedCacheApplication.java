package com.learning.cache.distributed;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;

@SpringBootApplication(scanBasePackages = {
    "com.learning.cache.distributed",
    "com.learning.cache.common"
})
@EntityScan(basePackages = "com.learning.cache.common.model")
@EnableJpaRepositories(basePackages = {
    "com.learning.cache.distributed.repository",
    "com.learning.cache.common.repository"
})
public class DistributedCacheApplication {

    public static void main(String[] args) {
        SpringApplication.run(DistributedCacheApplication.class, args);
    }
}
