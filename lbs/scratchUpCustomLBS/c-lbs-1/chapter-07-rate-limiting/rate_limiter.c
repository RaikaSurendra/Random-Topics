/**
 * Chapter 07: Rate Limiter Implementation
 * =======================================
 * Token Bucket and Sliding Window algorithms
 */

#include "rate_limiter.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#define HASH_SIZE 1024

static unsigned int hash_key(const char *key) {
    unsigned int hash = 5381;
    while (*key) {
        hash = ((hash << 5) + hash) + *key++;
    }
    return hash % HASH_SIZE;
}

rate_limiter_t* rate_limiter_create(RateLimitAlgorithm algorithm,
                                     double rate, double burst) {
    rate_limiter_t *limiter = calloc(1, sizeof(rate_limiter_t));
    if (!limiter) return NULL;

    limiter->algorithm = algorithm;
    limiter->rate = rate;
    limiter->burst = burst;
    limiter->window_size = (int)burst;
    limiter->num_buckets = HASH_SIZE;

    limiter->buckets = calloc(HASH_SIZE, sizeof(RateEntry*));
    if (!limiter->buckets) {
        free(limiter);
        return NULL;
    }

    pthread_mutex_init(&limiter->lock, NULL);
    return limiter;
}

void rate_limiter_destroy(rate_limiter_t *limiter) {
    if (!limiter) return;

    pthread_mutex_lock(&limiter->lock);

    for (int i = 0; i < limiter->num_buckets; i++) {
        RateEntry *entry = limiter->buckets[i];
        while (entry) {
            RateEntry *next = (RateEntry*)entry->next;
            free(entry);
            entry = next;
        }
    }

    free(limiter->buckets);
    pthread_mutex_unlock(&limiter->lock);
    pthread_mutex_destroy(&limiter->lock);
    free(limiter);
}

static RateEntry* find_or_create_entry(rate_limiter_t *limiter, const char *key) {
    unsigned int idx = hash_key(key);
    RateEntry *entry = limiter->buckets[idx];

    while (entry) {
        if (strcmp(entry->key, key) == 0) {
            return entry;
        }
        entry = (RateEntry*)entry->next;
    }

    // Create new entry
    entry = calloc(1, sizeof(RateEntry));
    if (!entry) return NULL;

    strncpy(entry->key, key, sizeof(entry->key) - 1);
    entry->tokens = limiter->burst;  // Start full
    entry->last_update = time(NULL);
    entry->window_start = time(NULL);

    entry->next = (struct RateEntry*)limiter->buckets[idx];
    limiter->buckets[idx] = entry;

    return entry;
}

static int token_bucket_allow(rate_limiter_t *limiter, RateEntry *entry) {
    time_t now = time(NULL);
    double elapsed = difftime(now, entry->last_update);

    // Refill tokens
    entry->tokens += elapsed * limiter->rate;
    if (entry->tokens > limiter->burst) {
        entry->tokens = limiter->burst;
    }
    entry->last_update = now;

    // Check if we have tokens
    if (entry->tokens >= 1.0) {
        entry->tokens -= 1.0;
        return 1;
    }

    return 0;
}

static int sliding_window_allow(rate_limiter_t *limiter, RateEntry *entry) {
    time_t now = time(NULL);

    // Check if window has passed
    if (now - entry->window_start >= limiter->window_size) {
        // Calculate weight of previous window
        double weight = 1.0 - ((double)(now - entry->window_start - limiter->window_size) /
                               limiter->window_size);
        if (weight < 0) weight = 0;

        // Weighted count from previous window
        long weighted_count = (long)(entry->window_count * weight);

        // Start new window
        entry->window_count = weighted_count;
        entry->window_start = now;
    }

    // Check rate
    double max_requests = limiter->rate * limiter->window_size;
    if (entry->window_count < max_requests) {
        entry->window_count++;
        return 1;
    }

    return 0;
}

static int fixed_window_allow(rate_limiter_t *limiter, RateEntry *entry) {
    time_t now = time(NULL);

    // Reset window if expired
    if (now - entry->window_start >= limiter->window_size) {
        entry->window_count = 0;
        entry->window_start = now;
    }

    // Check rate
    double max_requests = limiter->rate * limiter->window_size;
    if (entry->window_count < max_requests) {
        entry->window_count++;
        return 1;
    }

    return 0;
}

int rate_limiter_allow(rate_limiter_t *limiter, const char *key) {
    pthread_mutex_lock(&limiter->lock);

    // Check global limit
    if (limiter->global_limit > 0) {
        time_t now = time(NULL);
        if (now != limiter->global_window_start) {
            limiter->global_count = 0;
            limiter->global_window_start = now;
        }
        if (limiter->global_count >= limiter->global_limit) {
            limiter->denied++;
            pthread_mutex_unlock(&limiter->lock);
            return 0;
        }
    }

    RateEntry *entry = find_or_create_entry(limiter, key);
    if (!entry) {
        pthread_mutex_unlock(&limiter->lock);
        return 1;  // Allow on error
    }

    int allowed = 0;
    switch (limiter->algorithm) {
        case RATE_TOKEN_BUCKET:
            allowed = token_bucket_allow(limiter, entry);
            break;
        case RATE_SLIDING_WINDOW:
            allowed = sliding_window_allow(limiter, entry);
            break;
        case RATE_FIXED_WINDOW:
            allowed = fixed_window_allow(limiter, entry);
            break;
    }

    if (allowed) {
        limiter->allowed++;
        limiter->global_count++;
    } else {
        limiter->denied++;
    }

    pthread_mutex_unlock(&limiter->lock);
    return allowed;
}

void rate_limiter_set_global(rate_limiter_t *limiter, long limit) {
    pthread_mutex_lock(&limiter->lock);
    limiter->global_limit = limit;
    pthread_mutex_unlock(&limiter->lock);
}

double rate_limiter_remaining(rate_limiter_t *limiter, const char *key) {
    pthread_mutex_lock(&limiter->lock);

    unsigned int idx = hash_key(key);
    RateEntry *entry = limiter->buckets[idx];

    while (entry) {
        if (strcmp(entry->key, key) == 0) {
            double remaining = 0;
            switch (limiter->algorithm) {
                case RATE_TOKEN_BUCKET:
                    remaining = entry->tokens;
                    break;
                case RATE_SLIDING_WINDOW:
                case RATE_FIXED_WINDOW:
                    remaining = limiter->rate * limiter->window_size - entry->window_count;
                    break;
            }
            pthread_mutex_unlock(&limiter->lock);
            return remaining;
        }
        entry = (RateEntry*)entry->next;
    }

    pthread_mutex_unlock(&limiter->lock);
    return limiter->burst;  // Not found = full quota
}

void rate_limiter_get_stats(rate_limiter_t *limiter, rate_limiter_stats_t *stats) {
    pthread_mutex_lock(&limiter->lock);

    stats->allowed = limiter->allowed;
    stats->denied = limiter->denied;

    unsigned long total = stats->allowed + stats->denied;
    stats->denial_rate = total > 0 ? (double)stats->denied / total * 100.0 : 0;

    // Count active clients
    stats->active_clients = 0;
    for (int i = 0; i < limiter->num_buckets; i++) {
        RateEntry *entry = limiter->buckets[i];
        while (entry) {
            stats->active_clients++;
            entry = (RateEntry*)entry->next;
        }
    }

    pthread_mutex_unlock(&limiter->lock);
}

int rate_limiter_cleanup(rate_limiter_t *limiter) {
    pthread_mutex_lock(&limiter->lock);

    int cleaned = 0;
    time_t now = time(NULL);
    time_t expiry = 300;  // 5 minutes idle = cleanup

    for (int i = 0; i < limiter->num_buckets; i++) {
        RateEntry **prev = &limiter->buckets[i];
        RateEntry *entry = *prev;

        while (entry) {
            if (now - entry->last_update > expiry) {
                *prev = (RateEntry*)entry->next;
                free(entry);
                entry = *prev;
                cleaned++;
            } else {
                prev = (RateEntry**)&entry->next;
                entry = (RateEntry*)entry->next;
            }
        }
    }

    pthread_mutex_unlock(&limiter->lock);
    return cleaned;
}
