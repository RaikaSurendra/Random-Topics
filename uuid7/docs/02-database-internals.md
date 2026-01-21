# Database Internals: Pages, Heap, and B-Tree Indexes

Understanding how databases store and retrieve data is crucial for understanding why UUID choice matters.

## The Page: Fundamental Storage Unit

A **page** (also called a block) is the smallest unit of data that a database reads from or writes to disk.

### PostgreSQL Page Structure (8KB default)

```
+---------------------------+
|      Page Header          |  (24 bytes)
|  - LSN, checksum, flags   |
+---------------------------+
|     Item Pointers         |  (4 bytes each)
|  (array growing down)     |
|         ↓                 |
+---------------------------+
|                           |
|      Free Space           |
|                           |
+---------------------------+
|         ↑                 |
|    Tuple Data             |
|  (growing upward)         |
+---------------------------+
|    Special Space          |  (index-specific)
+---------------------------+
```

### Key Page Concepts

| Concept | Description |
|---------|-------------|
| **Page Size** | Fixed size (8KB PostgreSQL, 16KB MySQL InnoDB) |
| **Fill Factor** | Target percentage of page to fill (default 90-100%) |
| **Page Split** | When a page is full and needs to accommodate new data |
| **Buffer Pool** | In-memory cache of frequently accessed pages |

## The Heap: Where Table Data Lives

The **heap** is an unordered collection of pages containing actual table row data.

```
Table "users"
+--------+--------+--------+--------+--------+
| Page 0 | Page 1 | Page 2 | Page 3 | Page 4 |  ...
+--------+--------+--------+--------+--------+
    |        |        |        |        |
    v        v        v        v        v
 [rows]   [rows]   [rows]   [rows]   [rows]

Rows are stored wherever there's space - NO ordering guarantee
```

### Heap Characteristics
- **Unordered storage**: Rows go wherever there's free space
- **Fast inserts**: Just find any page with space
- **Sequential scans**: Must read all pages for full table scan
- **No inherent index**: Requires separate index structures

## B-Tree Index: The Performance Engine

A **B-Tree** (Balanced Tree) is the most common index structure, optimized for sorted data and range queries.

### B-Tree Structure

```
                        [Root Page]
                     [30 | 60 | 90]
                    /    |    |    \
                   /     |    |     \
            [Internal] [Internal] [Internal] [Internal]
            [10|20]    [40|50]    [70|80]    [100|110]
             /|\        /|\        /|\         /|\
            / | \      / | \      / | \       / | \
         [Leaf] [Leaf] [Leaf] [Leaf] [Leaf] [Leaf] ...

Each leaf contains: [key, pointer to heap tuple]
```

### B-Tree Properties

1. **Balanced**: All leaf nodes at same depth
2. **Sorted**: Keys in each node are sorted
3. **Fan-out**: Each node contains many keys (high branching factor)
4. **Logarithmic search**: O(log n) lookups

### Page Layout in B-Tree Index

```
Index Page (8KB)
+---------------------------+
|      Page Header          |
+---------------------------+
| Key1 → (PageID, Offset)   |
| Key2 → (PageID, Offset)   |
| Key3 → (PageID, Offset)   |
|           ...             |
| KeyN → (PageID, Offset)   |
+---------------------------+
| Child page pointers       |
| (for non-leaf pages)      |
+---------------------------+
```

## How B-Tree Insertion Works

### Sequential Keys (e.g., AUTO_INCREMENT, UUIDv7)

```
Insert: 1, 2, 3, 4, 5, 6, 7, 8...

Step 1: [1, 2, 3] - Fill leaf page
Step 2: [1, 2, 3, 4] - Continue filling
Step 3: [1, 2, 3, 4, 5] - Page full, split RIGHT

        [3]              ← New root
       /   \
    [1,2]  [3,4,5]       ← Always inserting to rightmost leaf

New inserts go to the RIGHT-most leaf (hot spot is predictable)
```

**Benefits:**
- New inserts always go to the rightmost page
- That page stays in buffer pool (memory)
- Minimal page splits
- Excellent cache locality

### Random Keys (e.g., UUIDv4)

```
Insert: f47a..., 12ab..., 89cd..., 3456...

The tree after many random inserts:

              [5678...]
            /          \
    [1234...3456...]  [5678...f47a...]
      /    |    \        /    |    \
   [12ab] [2345] [3456] [5678] [89cd] [f47a]
     ↑      ↑      ↑      ↑      ↑      ↑
     └──────┴──────┴──────┴──────┴──────┘
        Random inserts touch ALL pages!
```

**Problems:**
- Inserts scattered across entire tree
- Every insert might need different page
- Constant page cache misses
- Frequent page splits in the MIDDLE of tree

## Page Splits: The Hidden Performance Killer

### What is a Page Split?

When a page is full and a new key must be inserted into it, the database must:

1. Allocate a new page
2. Move ~50% of entries to new page
3. Update parent pointers
4. Write multiple pages to disk

```
Before Split:                After Split:
+-------------+              +-------------+  +-------------+
| A B C D E F |    →         | A B C |      | D E F [new] |
+-------------+              +-------------+  +-------------+
     (full)                       ↓               ↓
                            Update parent to point to both
```

### Page Split Impact by Key Type

| Metric | Sequential (UUIDv7) | Random (UUIDv4) |
|--------|---------------------|-----------------|
| Split Location | Right edge only | Throughout tree |
| Pages Affected | 1-2 per split | 2-3 per split |
| Split Frequency | Low | High |
| Write Amplification | Low | High |
| Index Fragmentation | Minimal | Severe |

## Buffer Pool and Cache Efficiency

The **buffer pool** is the database's in-memory cache of pages.

```
                    RAM (Buffer Pool)
+-------------------------------------------------------+
|  [Page 42] [Page 17] [Page 99] [Page 3] [Page 156]   |
|    HOT       HOT       WARM      COLD      COLD       |
+-------------------------------------------------------+
        ↑                                    ↓
   Frequently                          Evicted when
    accessed                           memory needed
```

### Working Set Comparison

**UUIDv7 Workload:**
```
Inserts only touch rightmost pages → Small working set
[Page N-2] [Page N-1] [Page N] ← Only these in buffer pool
```

**UUIDv4 Workload:**
```
Inserts touch random pages → Entire index = working set
[Page 1] [Page 2] ... [Page N] ← ALL pages needed
```

### Real-World Impact

For a table with 10 million rows and a UUID primary key:

| Scenario | Index Size | UUIDv7 Working Set | UUIDv4 Working Set |
|----------|------------|--------------------|--------------------|
| Inserts | ~400MB | ~24KB (3 pages) | ~400MB (entire index) |
| Buffer Pool (1GB) | - | Index fits easily | Index fits |
| Buffer Pool (256MB) | - | Index fits easily | Constant evictions |

## Index Fragmentation

Over time, random inserts cause **fragmentation**:

```
Fragmented Index (UUIDv4 after many inserts):

Page 1: [a1...] [____] [c3...] [____] [e5...]  ← 40% full
Page 2: [b2...] [____] [____] [d4...] [____]  ← 30% full
Page 3: [____] [f6...] [____] [____] [g7...]  ← 25% full

Wasted space = more pages = more I/O = slower queries
```

**Defragmentation requires:**
- REINDEX operation
- Table downtime or heavy locking
- Significant I/O

## Summary: Why Key Choice Matters

```
Sequential Keys (UUIDv7)          Random Keys (UUIDv4)
        │                                 │
        ▼                                 ▼
  Append to end                    Insert anywhere
        │                                 │
        ▼                                 ▼
 1-2 pages in cache              Entire index accessed
        │                                 │
        ▼                                 ▼
  Rare page splits              Frequent page splits
        │                                 │
        ▼                                 ▼
 Low fragmentation              High fragmentation
        │                                 │
        ▼                                 ▼
   HIGH PERFORMANCE               POOR PERFORMANCE
```
