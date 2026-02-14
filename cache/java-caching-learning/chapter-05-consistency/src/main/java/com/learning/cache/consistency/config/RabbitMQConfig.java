package com.learning.cache.consistency.config;

import org.springframework.amqp.core.*;
import org.springframework.amqp.rabbit.connection.ConnectionFactory;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.amqp.support.converter.Jackson2JsonMessageConverter;
import org.springframework.amqp.support.converter.MessageConverter;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class RabbitMQConfig {

    public static final String CACHE_INVALIDATION_EXCHANGE = "cache.invalidation.exchange";
    public static final String CACHE_INVALIDATION_QUEUE = "cache.invalidation.queue";
    public static final String CACHE_INVALIDATION_ROUTING_KEY = "cache.invalidation.#";

    @Bean
    public FanoutExchange cacheInvalidationExchange() {
        return new FanoutExchange(CACHE_INVALIDATION_EXCHANGE, true, false);
    }

    @Bean
    public Queue cacheInvalidationQueue() {
        return QueueBuilder.durable(CACHE_INVALIDATION_QUEUE)
                .withArgument("x-message-ttl", 60000)
                .build();
    }

    @Bean
    public Binding cacheInvalidationBinding(Queue cacheInvalidationQueue,
                                            FanoutExchange cacheInvalidationExchange) {
        return BindingBuilder.bind(cacheInvalidationQueue)
                .to(cacheInvalidationExchange);
    }

    @Bean
    public MessageConverter jsonMessageConverter() {
        return new Jackson2JsonMessageConverter();
    }

    @Bean
    public RabbitTemplate rabbitTemplate(ConnectionFactory connectionFactory,
                                         MessageConverter messageConverter) {
        RabbitTemplate template = new RabbitTemplate(connectionFactory);
        template.setMessageConverter(messageConverter);
        return template;
    }
}
