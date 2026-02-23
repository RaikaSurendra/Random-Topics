# Chapter 05: Attention & The Transformer Architecture

## The Key Insight

The MLP from Chapter 04 processes each token **independently**. But language requires **context** — the meaning of "bank" depends on whether we're talking about rivers or money.

**Attention** lets each token look at all previous tokens and decide which ones are relevant.

---

## Attention in Plain English

Imagine you're reading: "The cat sat on the ___"

To predict the next word, you need to look back at the whole sentence. But not all words matter equally:
- "cat" is very relevant (what's sitting?)
- "The" is less relevant
- "sat on" tells you it's about a location

Attention is a mechanism that **automatically learns** which previous tokens to focus on.

---

## How Attention Works: Query, Key, Value

Each token produces three vectors:
- **Query (Q)**: "What am I looking for?"
- **Key (K)**: "What do I contain?"
- **Value (V)**: "What information do I provide?"

The process:
1. Each token's Query is compared against all previous tokens' Keys (dot product)
2. The dot products become **attention weights** (via softmax)
3. The weighted sum of Values is the output

```
Token "___" asks: "Who should I pay attention to?"
  - Q("___") · K("cat")  = 0.7  (high match!)
  - Q("___") · K("The")  = 0.1  (low match)
  - Q("___") · K("sat")  = 0.5  (medium match)
  - Q("___") · K("on")   = 0.6  (medium-high match)
  - Q("___") · K("the")  = 0.1  (low match)

After softmax: weights = [0.35, 0.05, 0.18, 0.27, 0.05]
Output = weighted sum of all Value vectors
```

---

## Multi-Head Attention

Instead of one set of Q/K/V, GPT uses **multiple heads** (e.g., 4 or 12). Each head can learn to pay attention to different things:
- Head 1 might focus on syntax (subject-verb agreement)
- Head 2 might focus on nearby context
- Head 3 might focus on semantic similarity

The outputs of all heads are concatenated.

---

## Causal (Masked) Attention

GPT is **autoregressive** — it generates text left to right. So token at position 5 can only attend to positions 0-4, never to future positions. This is enforced by masking.

---

## The Transformer Block

One transformer block = Attention + MLP, with residual connections and normalization:

```
Input
  │
  ├──→ Norm → Multi-Head Attention ──┐
  │                                   │
  └───────────────────── (+) ←────────┘  ← Residual connection
  │
  ├──→ Norm → MLP (Linear → ReLU → Linear) ──┐
  │                                             │
  └─────────────────────────────── (+) ←────────┘  ← Residual connection
  │
Output
```

GPT stacks multiple blocks: microGPT uses 1, GPT-2 uses 12-48.

---

## Run the Examples

```bash
python attention_basics.py       # Step-by-step attention computation
python multi_head_attention.py   # Multi-head attention from scratch
python transformer_block.py      # A complete transformer block
```
