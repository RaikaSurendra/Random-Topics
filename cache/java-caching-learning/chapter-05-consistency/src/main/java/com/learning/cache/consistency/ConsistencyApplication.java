package com.learning.cache.consistency;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;
import org.springframework.scheduling.annotation.EnableAsync;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication(scanBasePackages = {
    "com.learning.cache.consistency",
    "com.learning.cache.common"
})
@EnableAsync
@EnableScheduling
@EntityScan(basePackages = "com.learning.cache.common.model")
@EnableJpaRepositories(basePackages = {
    "com.learning.cache.consistency.repository",
    "com.learning.cache.common.repository"
})
public class ConsistencyApplication {

    public static void main(String[] args) {
        SpringApplication.run(ConsistencyApplication.class, args);
    }
}
