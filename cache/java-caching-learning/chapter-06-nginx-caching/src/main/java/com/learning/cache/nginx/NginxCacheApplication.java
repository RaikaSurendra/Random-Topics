package com.learning.cache.nginx;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.domain.EntityScan;
import org.springframework.data.jpa.repository.config.EnableJpaRepositories;

@SpringBootApplication(scanBasePackages = {
    "com.learning.cache.nginx",
    "com.learning.cache.common"
})
@EntityScan(basePackages = "com.learning.cache.common.model")
@EnableJpaRepositories(basePackages = {
    "com.learning.cache.nginx.repository",
    "com.learning.cache.common.repository"
})
public class NginxCacheApplication {

    public static void main(String[] args) {
        SpringApplication.run(NginxCacheApplication.class, args);
    }
}
