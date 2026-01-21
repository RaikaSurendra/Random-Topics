# Understanding UUIDs: Version 4 vs Version 7

## What is a UUID?

A **UUID** (Universally Unique Identifier) is a 128-bit identifier designed to be unique across space and time without requiring a central registration authority.

```
Format: xxxxxxxx-xxxx-Mxxx-Nxxx-xxxxxxxxxxxx
        |        |    |    |    |
        |        |    |    |    +-- Node (48 bits)
        |        |    |    +------- Clock sequence (13-14 bits)
        |        |    +------------ Version (M = version number)
        |        +----------------- Time (varies by version)
        +-------------------------- Time (varies by version)
```

## UUID Version 4 (Random)

UUIDv4 generates identifiers using **random or pseudo-random numbers**.

### Structure
```
xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
              ^    ^
              |    +-- Variant (8, 9, a, or b)
              +------- Version 4
```

### Characteristics
- **122 bits of randomness** (6 bits reserved for version/variant)
- No embedded timestamp
- No inherent ordering
- Collision probability: ~2.71 quintillion UUIDs needed for 50% collision chance

### Example
```
f47ac10b-58cc-4372-a567-0e02b2c3d479
a1b2c3d4-e5f6-4789-abcd-ef0123456789
9f8e7d6c-5b4a-4321-8765-432109876543
```

## UUID Version 7 (Time-Ordered)

UUIDv7 (RFC 9562, 2024) combines **Unix timestamp with randomness** for time-ordered unique identifiers.

### Structure
```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          unix_ts_ms                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           unix_ts_ms          |  ver  |       rand_a         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|var|                        rand_b                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                            rand_b                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### Components
| Field | Bits | Description |
|-------|------|-------------|
| unix_ts_ms | 48 | Milliseconds since Unix epoch |
| ver | 4 | Version (always 7) |
| rand_a | 12 | Random data |
| var | 2 | Variant (RFC 4122) |
| rand_b | 62 | Random data |

### Characteristics
- **Monotonically increasing** (time-ordered)
- 48-bit timestamp allows dates until year 10889
- 74 bits of randomness per millisecond
- Natural chronological sorting

### Example (generated sequentially)
```
018f6e7c-8a3b-7def-8123-456789abcdef  (earlier)
018f6e7c-8a3c-7012-9345-6789abcdef01  (later)
018f6e7c-8a3d-7234-a567-89abcdef0123  (latest)
         ^^^^
         These increase over time
```

## Quick Comparison

| Aspect | UUIDv4 | UUIDv7 |
|--------|--------|--------|
| Ordering | Random | Time-ordered |
| Timestamp | None | 48-bit ms precision |
| Randomness | 122 bits | 74 bits |
| Index Performance | Poor | Excellent |
| Cache Locality | Poor | Excellent |
| Sortable | No | Yes |
| Privacy | Better (no time leak) | Time extractable |

## When to Use Which?

### Use UUIDv4 when:
- Privacy is critical (no time information should leak)
- Order doesn't matter
- Working with legacy systems requiring UUIDv4
- Generating IDs client-side where time sync is unreliable

### Use UUIDv7 when:
- Database performance matters (most cases)
- Natural ordering by creation time is useful
- Using B-tree indexes (most databases)
- High-volume insert workloads
- You need both uniqueness AND sortability
