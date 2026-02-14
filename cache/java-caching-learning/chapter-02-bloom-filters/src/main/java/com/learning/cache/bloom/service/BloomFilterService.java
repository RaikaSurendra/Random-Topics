package com.learning.cache.bloom.service;

import java.util.Collection;

public interface BloomFilterService {

    boolean mightContain(String item);

    void add(String item);

    void addAll(Collection<String> items);

    void rebuild(Collection<String> items);

    void clear();

    long approximateElementCount();

    double expectedFalsePositiveRate();
}
