"""
Chapter 05: Multi-Head Attention

Instead of one attention mechanism, GPT uses MULTIPLE "heads"
that run in parallel. Each head can learn to focus on different
types of relationships.

This code follows the exact pattern from microGPT (microgpt.py).
"""

import math
import random

random.seed(42)

# ============================================================
# Configuration (matches microGPT's tiny config)
# ============================================================
n_embd = 8       # embedding dimension
n_head = 2       # number of attention heads
head_dim = n_embd // n_head  # = 4 dimensions per head
seq_len = 4      # sequence length

print("=" * 60)
print("Multi-Head Attention")
print("=" * 60)
print(f"  Embedding dim (n_embd): {n_embd}")
print(f"  Number of heads:        {n_head}")
print(f"  Dimension per head:     {head_dim}")
print(f"  Sequence length:        {seq_len}")

# ============================================================
# Initialize weight matrices
# ============================================================
def make_matrix(rows, cols):
    return [[random.gauss(0, 0.3) for _ in range(cols)] for _ in range(rows)]

Wq = make_matrix(n_embd, n_embd)  # Query projection
Wk = make_matrix(n_embd, n_embd)  # Key projection
Wv = make_matrix(n_embd, n_embd)  # Value projection
Wo = make_matrix(n_embd, n_embd)  # Output projection

def linear(x, w):
    return [sum(wi * xi for wi, xi in zip(row, x)) for row in w]

def softmax(logits):
    max_val = max(logits)
    exps = [math.exp(v - max_val) for v in logits]
    total = sum(exps)
    return [e / total for e in exps]

# ============================================================
# Generate token embeddings (pretend these came from earlier layers)
# ============================================================
X = [[random.gauss(0, 0.5) for _ in range(n_embd)] for _ in range(seq_len)]

# ============================================================
# Multi-Head Attention (KV-cache style, like microGPT)
# ============================================================
# microGPT processes one token at a time, accumulating keys/values
# This is more memory efficient and matches how inference works.

print("\n--- Processing tokens one at a time (KV-cache style) ---\n")

all_keys = []   # accumulated keys
all_values = []  # accumulated values
outputs = []

for pos in range(seq_len):
    x = X[pos]

    # Project current token to Q, K, V (full n_embd dimension)
    q = linear(x, Wq)
    k = linear(x, Wk)
    v = linear(x, Wv)

    # Store K and V for future tokens to attend to
    all_keys.append(k)
    all_values.append(v)

    # Now run each attention head
    x_attn = []  # will collect outputs from all heads

    for h in range(n_head):
        # Each head gets a SLICE of the Q, K, V vectors
        hs = h * head_dim  # start index for this head

        # Slice out this head's portion
        q_h = q[hs:hs + head_dim]
        k_h = [ki[hs:hs + head_dim] for ki in all_keys]    # all stored keys
        v_h = [vi[hs:hs + head_dim] for vi in all_values]   # all stored values

        # Compute attention scores: Q dot K for all positions up to current
        attn_logits = []
        for t in range(len(k_h)):
            dot = sum(q_h[j] * k_h[t][j] for j in range(head_dim))
            attn_logits.append(dot / head_dim ** 0.5)

        # Softmax to get weights
        attn_weights = softmax(attn_logits)

        # Weighted sum of values
        head_out = []
        for j in range(head_dim):
            val = sum(attn_weights[t] * v_h[t][j] for t in range(len(v_h)))
            head_out.append(val)

        x_attn.extend(head_out)  # Concatenate head outputs

        if pos == seq_len - 1:  # Print details for last token
            print(f"  Token {pos}, Head {h}:")
            print(f"    Attention weights: [{', '.join(f'{w:.3f}' for w in attn_weights)}]")
            print(f"    Head output: [{', '.join(f'{v:.3f}' for v in head_out)}]")

    # Final output projection: combine all heads
    out = linear(x_attn, Wo)
    outputs.append(out)

print(f"\n--- Outputs ---")
for pos in range(seq_len):
    print(f"  Token {pos}: [{', '.join(f'{v:+.3f}' for v in outputs[pos][:4])}...]")

# ============================================================
# Explain Multi-Head
# ============================================================
print(f"""
=== Why Multiple Heads? ===

With {n_head} heads, each getting {head_dim} dimensions:
  - Head 0 uses dimensions [0:{head_dim}] of Q, K, V
  - Head 1 uses dimensions [{head_dim}:{n_embd}] of Q, K, V

Each head learns DIFFERENT attention patterns:
  - Head 0 might learn to attend to the previous token
  - Head 1 might learn to attend to the subject of the sentence

After computing, head outputs are CONCATENATED:
  [{head_dim} dims from head 0] + [{head_dim} dims from head 1] = [{n_embd} dims total]

Then a final linear layer (Wo) mixes the head outputs.

=== KV-Cache ===

Notice how we process ONE token at a time and store its K, V:
  - Token 0: compute K0, V0, store them
  - Token 1: compute K1, V1, attend to [K0, K1], [V0, V1]
  - Token 2: compute K2, V2, attend to [K0, K1, K2], [V0, V1, V2]

This is exactly how microGPT works (and how real GPT inference works).
The "KV-cache" avoids recomputing keys and values for earlier tokens.
""")
