"""
Chapter 05: Attention — Step by Step

This builds single-head attention from scratch, showing every
intermediate value so you can see exactly what's happening.
"""

import math
import random

random.seed(42)

# ============================================================
# SETUP: A tiny sequence of 3 tokens, each with 4-dimensional embeddings
# ============================================================
# Imagine we're processing: "the cat sat"
# After embedding, each token is a vector of numbers.

seq_len = 3
d_model = 4  # embedding dimension

# Pretend these are embeddings (normally learned, here random for demo)
tokens = {
    0: "the",
    1: "cat",
    2: "sat",
}

# Token embeddings (what the model "sees")
X = [
    [1.0, 0.5, -0.3, 0.8],   # "the"
    [0.2, 1.2,  0.7, -0.1],   # "cat"
    [-0.5, 0.3, 1.0,  0.6],   # "sat"
]

print("=" * 60)
print("Single-Head Self-Attention — Step by Step")
print("=" * 60)
print(f"\nSequence: {[tokens[i] for i in range(seq_len)]}")
print(f"Embedding dimension: {d_model}")
print(f"\nToken embeddings (X):")
for i in range(seq_len):
    print(f"  {tokens[i]:>5}: {X[i]}")

# ============================================================
# STEP 1: Project to Q, K, V using weight matrices
# ============================================================
# In practice, these are learned weight matrices.
# Here we use simple random values for illustration.

head_dim = d_model  # In multi-head, this would be d_model // n_heads

# Weight matrices (would be learned during training)
Wq = [[random.gauss(0, 0.5) for _ in range(d_model)] for _ in range(head_dim)]
Wk = [[random.gauss(0, 0.5) for _ in range(d_model)] for _ in range(head_dim)]
Wv = [[random.gauss(0, 0.5) for _ in range(d_model)] for _ in range(head_dim)]

def matmul_vec(w, x):
    """Multiply weight matrix by input vector."""
    return [sum(wi * xi for wi, xi in zip(row, x)) for row in w]

# Compute Q, K, V for each token
Q = [matmul_vec(Wq, X[i]) for i in range(seq_len)]
K = [matmul_vec(Wk, X[i]) for i in range(seq_len)]
V = [matmul_vec(Wv, X[i]) for i in range(seq_len)]

print("\n--- Step 1: Compute Q (Query), K (Key), V (Value) ---")
for i in range(seq_len):
    print(f"  {tokens[i]:>5} Q: [{', '.join(f'{v:+.2f}' for v in Q[i])}]")
    print(f"  {tokens[i]:>5} K: [{', '.join(f'{v:+.2f}' for v in K[i])}]")
    print(f"  {tokens[i]:>5} V: [{', '.join(f'{v:+.2f}' for v in V[i])}]")

# ============================================================
# STEP 2: Compute attention scores (Q dot K)
# ============================================================
print("\n--- Step 2: Attention Scores = Q · K^T / sqrt(d) ---")
print("  Each score measures how much token i 'wants to attend to' token j.\n")

scale = math.sqrt(head_dim)
scores = [[0.0] * seq_len for _ in range(seq_len)]

for i in range(seq_len):
    for j in range(seq_len):
        # Dot product of Q[i] and K[j]
        dot = sum(Q[i][d] * K[j][d] for d in range(head_dim))
        scores[i][j] = dot / scale

# Print score matrix
print(f"  {'':>10}", end="")
for j in range(seq_len):
    print(f"{tokens[j]:>10}", end="")
print()
for i in range(seq_len):
    print(f"  {tokens[i]:>10}", end="")
    for j in range(seq_len):
        print(f"{scores[i][j]:>10.3f}", end="")
    print()

# ============================================================
# STEP 3: Apply CAUSAL MASK (can't look at future tokens)
# ============================================================
print("\n--- Step 3: Apply Causal Mask (hide future) ---")
print("  GPT generates left-to-right, so each token can only see itself and earlier tokens.\n")

masked_scores = [[0.0] * seq_len for _ in range(seq_len)]
for i in range(seq_len):
    for j in range(seq_len):
        if j > i:  # future token — mask it!
            masked_scores[i][j] = float('-inf')
        else:
            masked_scores[i][j] = scores[i][j]

print(f"  {'':>10}", end="")
for j in range(seq_len):
    print(f"{tokens[j]:>10}", end="")
print()
for i in range(seq_len):
    print(f"  {tokens[i]:>10}", end="")
    for j in range(seq_len):
        if masked_scores[i][j] == float('-inf'):
            print(f"{'  -inf':>10}", end="")
        else:
            print(f"{masked_scores[i][j]:>10.3f}", end="")
    print()

# ============================================================
# STEP 4: Softmax → attention weights (probabilities)
# ============================================================
print("\n--- Step 4: Softmax → Attention Weights ---")
print("  Convert scores to probabilities. -inf becomes 0 (masked out).\n")

def softmax(logits):
    max_val = max(v for v in logits if v != float('-inf'))
    exps = [math.exp(v - max_val) if v != float('-inf') else 0.0 for v in logits]
    total = sum(exps)
    return [e / total if total > 0 else 0.0 for e in exps]

attn_weights = [softmax(masked_scores[i]) for i in range(seq_len)]

print(f"  {'':>10}", end="")
for j in range(seq_len):
    print(f"{tokens[j]:>10}", end="")
print()
for i in range(seq_len):
    print(f"  {tokens[i]:>10}", end="")
    for j in range(seq_len):
        print(f"{attn_weights[i][j]:>10.3f}", end="")
    print(f"  (sum={sum(attn_weights[i]):.3f})")

print("\n  Interpretation:")
print(f"    '{tokens[0]}' attends to: only itself (first token, nothing before it)")
print(f"    '{tokens[1]}' attends to: '{tokens[0]}' and itself")
print(f"    '{tokens[2]}' attends to: all three tokens")

# ============================================================
# STEP 5: Weighted sum of Values → output
# ============================================================
print("\n--- Step 5: Output = Attention Weights × Values ---")
print("  Each token's output is a weighted blend of all attended Value vectors.\n")

output = []
for i in range(seq_len):
    out_i = [0.0] * head_dim
    for j in range(seq_len):
        for d in range(head_dim):
            out_i[d] += attn_weights[i][j] * V[j][d]
    output.append(out_i)

for i in range(seq_len):
    print(f"  {tokens[i]:>5} output: [{', '.join(f'{v:+.3f}' for v in output[i])}]")

print("""
=== Summary ===
Attention in 5 steps:
  1. Project each token to Q, K, V vectors
  2. Score: how much does each Q match each K? (dot product)
  3. Mask: hide future tokens (causal / autoregressive)
  4. Softmax: convert scores to weights (probabilities)
  5. Output: weighted sum of V vectors

Key insight: The model LEARNS the W_q, W_k, W_v matrices during training.
After training, the attention patterns emerge automatically — the model
discovers which tokens are relevant to which other tokens.
""")
