package com.learning.cache.race;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;
import org.springframework.scheduling.annotation.EnableAsync;

@SpringBootApplication(scanBasePackages = {
    "com.learning.cache.race",
    "com.learning.cache.common"
})
@EnableAsync
@EntityScan(basePackages = "com.learning.cache.common.model")
@EnableJpaRepositories(basePackages = {
    "com.learning.cache.race.repository",
    "com.learning.cache.common.repository"
})
public class RaceConditionApplication {

    public static void main(String[] args) {
        SpringApplication.run(RaceConditionApplication.class, args);
    }
}
