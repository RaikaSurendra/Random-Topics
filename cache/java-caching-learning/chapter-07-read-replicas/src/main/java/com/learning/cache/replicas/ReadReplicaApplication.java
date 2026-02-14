package com.learning.cache.replicas;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;

@SpringBootApplication(scanBasePackages = {
    "com.learning.cache.replicas",
    "com.learning.cache.common"
})
@EntityScan(basePackages = "com.learning.cache.common.model")
@EnableJpaRepositories(basePackages = {
    "com.learning.cache.replicas.repository",
    "com.learning.cache.common.repository"
})
public class ReadReplicaApplication {

    public static void main(String[] args) {
        SpringApplication.run(ReadReplicaApplication.class, args);
    }
}
