package com.learning.cache.bloom.service;

import com.google.common.hash.BloomFilter;
import com.google.common.hash.Funnels;
import lombok.extern.slf4j.Slf4j;

import java.nio.charset.StandardCharsets;
import java.util.Collection;
import java.util.concurrent.locks.ReentrantReadWriteLock;

@Slf4j
public class InMemoryBloomFilter implements BloomFilterService {

    private final int expectedInsertions;
    private final double falsePositiveRate;
    private final ReentrantReadWriteLock lock = new ReentrantReadWriteLock();

    private volatile BloomFilter<String> filter;

    public InMemoryBloomFilter(int expectedInsertions, double falsePositiveRate) {
        this.expectedInsertions = expectedInsertions;
        this.falsePositiveRate = falsePositiveRate;
        this.filter = createFilter();
        log.info("Created in-memory Bloom filter: expectedInsertions={}, falsePositiveRate={}",
                expectedInsertions, falsePositiveRate);
    }

    @Override
    public boolean mightContain(String item) {
        lock.readLock().lock();
        try {
            return filter.mightContain(item);
        } finally {
            lock.readLock().unlock();
        }
    }

    @Override
    public void add(String item) {
        lock.writeLock().lock();
        try {
            filter.put(item);
            log.debug("Added item to Bloom filter: {}", item);
        } finally {
            lock.writeLock().unlock();
        }
    }

    @Override
    public void addAll(Collection<String> items) {
        lock.writeLock().lock();
        try {
            items.forEach(filter::put);
            log.info("Added {} items to Bloom filter", items.size());
        } finally {
            lock.writeLock().unlock();
        }
    }

    @Override
    public void rebuild(Collection<String> items) {
        log.info("Rebuilding Bloom filter with {} items", items.size());
        BloomFilter<String> newFilter = createFilter();
        items.forEach(newFilter::put);

        lock.writeLock().lock();
        try {
            this.filter = newFilter;
        } finally {
            lock.writeLock().unlock();
        }
        log.info("Bloom filter rebuild complete");
    }

    @Override
    public void clear() {
        lock.writeLock().lock();
        try {
            this.filter = createFilter();
            log.info("Bloom filter cleared");
        } finally {
            lock.writeLock().unlock();
        }
    }

    @Override
    public long approximateElementCount() {
        lock.readLock().lock();
        try {
            return filter.approximateElementCount();
        } finally {
            lock.readLock().unlock();
        }
    }

    @Override
    public double expectedFalsePositiveRate() {
        lock.readLock().lock();
        try {
            return filter.expectedFpp();
        } finally {
            lock.readLock().unlock();
        }
    }

    private BloomFilter<String> createFilter() {
        return BloomFilter.create(
                Funnels.stringFunnel(StandardCharsets.UTF_8),
                expectedInsertions,
                falsePositiveRate
        );
    }
}
