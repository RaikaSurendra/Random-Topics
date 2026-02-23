# Chapter 02: Tokenization — Turning Text into Numbers

## Why Tokenization?

Computers don't understand letters or words — they work with **numbers**. Before we can feed text into any model, we need to convert it to a sequence of integers. This conversion is called **tokenization**.

```
"hello" → [7, 4, 11, 11, 14]
```

---

## Three Levels of Tokenization

### 1. Character-Level (used by microGPT)

Each unique character gets a number:

```
Vocabulary: {a:0, b:1, c:2, d:3, e:4, ...}
"cab" → [2, 0, 1]
```

**Pros:** Tiny vocabulary, simple to implement
**Cons:** Sequences are long, model must learn to spell words

### 2. Word-Level

Each unique word gets a number:

```
Vocabulary: {"the":0, "cat":1, "sat":2, ...}
"the cat" → [0, 1]
```

**Pros:** Short sequences
**Cons:** Huge vocabulary, can't handle new/misspelled words

### 3. Subword / BPE (used by GPT-2, minGPT)

Byte Pair Encoding — a clever middle ground. Frequent words stay whole, rare words get split:

```
"unhappiness" → ["un", "happiness"]  (common prefix + common word)
"Karpathy"    → ["K", "arp", "athy"] (rare name, split into pieces)
```

**Pros:** Handles any text, balanced vocabulary size
**Cons:** More complex to implement

---

## The Vocabulary

The set of all possible tokens is the **vocabulary**. Its size matters:

| Tokenizer | Vocab Size | Example |
|---|---|---|
| microGPT (names dataset) | ~27 | a-z + BOS token |
| GPT-2 (BPE) | 50,257 | All common English subwords |

The model's final layer must output a probability for **every token in the vocabulary**. Bigger vocab = bigger model.

---

## Special Tokens

Most tokenizers have special tokens with reserved meanings:

- **BOS** (Beginning of Sequence): Marks the start of a document
- **EOS** (End of Sequence): Marks the end
- **PAD**: Fills empty space when batching sequences of different lengths

In microGPT, there's one special token: `BOS` (used for both start and end).

---

## Run the Examples

```bash
python char_tokenizer.py     # Character-level tokenization (like microGPT)
python bpe_intuition.py      # How BPE works, step by step
```
