package com.learning.cache.common.util;

import lombok.extern.slf4j.Slf4j;

import java.util.concurrent.ThreadLocalRandom;

@Slf4j
public class SimulatedDelay {

    public static void databaseQuery() {
        databaseQuery(20, 50);
    }

    public static void databaseQuery(int minMs, int maxMs) {
        sleep(randomBetween(minMs, maxMs));
    }

    public static void networkCall() {
        networkCall(5, 15);
    }

    public static void networkCall(int minMs, int maxMs) {
        sleep(randomBetween(minMs, maxMs));
    }

    public static void heavyComputation() {
        heavyComputation(50, 100);
    }

    public static void heavyComputation(int minMs, int maxMs) {
        sleep(randomBetween(minMs, maxMs));
    }

    public static void sleep(long millis) {
        try {
            Thread.sleep(millis);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            log.warn("Sleep interrupted");
        }
    }

    private static int randomBetween(int min, int max) {
        return ThreadLocalRandom.current().nextInt(min, max + 1);
    }
}
