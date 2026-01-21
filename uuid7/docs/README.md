# UUID in Databases: A Deep Dive

Understanding why UUID choice matters for database performance.

## Documentation Structure

```
docs/
├── 01-uuid-overview.md          # UUID versions explained
├── 02-database-internals.md     # Pages, heap, B-tree concepts
├── 03-performance-analysis.md   # UUIDv4 vs UUIDv7 deep dive
├── 04-best-practices.md         # Implementation guidelines
└── README.md                    # This file

example/
├── docker-compose.yml           # PostgreSQL 17 setup
├── init/01-setup.sql            # Table definitions
├── benchmark.sql                # Performance tests
├── run-benchmark.sh             # Easy runner script
└── README.md                    # Example instructions
```

## Reading Order

1. **[UUID Overview](01-uuid-overview.md)** - Start here to understand UUIDv4 vs UUIDv7
2. **[Database Internals](02-database-internals.md)** - Learn about pages, heap, and B-trees
3. **[Performance Analysis](03-performance-analysis.md)** - Understand why UUIDv4 hurts performance
4. **[Best Practices](04-best-practices.md)** - Implementation recommendations

## Quick Summary

### The Problem with UUIDv4

```
UUIDv4 = 122 bits of randomness
       = Inserts scattered across entire B-tree
       = Poor cache locality
       = Frequent page splits
       = Index fragmentation
       = SLOW
```

### The Solution: UUIDv7

```
UUIDv7 = 48-bit timestamp + 74 bits randomness
       = Time-ordered inserts
       = Append to rightmost leaf
       = Excellent cache locality
       = Minimal page splits
       = FAST
```

### Visual Comparison

```
                 UUIDv4                          UUIDv7

Insert Pattern:  [Random across tree]            [Append to end]
Cache Needed:    [Entire index]                  [Last few pages]
Page Splits:     [Throughout tree]               [Right edge only]
Performance:     [Degrades with size]            [Stays consistent]
```

## Running the Benchmark

```bash
cd example/
./run-benchmark.sh start     # Start PostgreSQL
./run-benchmark.sh benchmark # Run tests
./run-benchmark.sh clean     # Cleanup
```

## Key Takeaways

1. **Always prefer UUIDv7 over UUIDv4** for database primary keys
2. **BIGSERIAL is still fastest** if you don't need distributed IDs
3. **UUIDv7 enables BRIN indexes** - tiny indexes for huge tables
4. **Consider hybrid approach** - BIGSERIAL internally, UUID externally
5. **The difference scales** - larger tables = bigger performance gap
