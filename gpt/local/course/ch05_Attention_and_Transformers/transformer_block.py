"""
Chapter 05: A Complete Transformer Block

This combines everything from Chapters 04 and 05:
  - RMSNorm
  - Multi-Head Attention
  - MLP (Linear → ReLU → Linear)
  - Residual Connections

This is the EXACT structure of one layer in microGPT.
"""

import math
import random

random.seed(42)

# ============================================================
# Configuration
# ============================================================
n_embd = 8
n_head = 2
head_dim = n_embd // n_head

# ============================================================
# Helper functions (from earlier chapters)
# ============================================================
def linear(x, w):
    return [sum(wi * xi for wi, xi in zip(row, x)) for row in w]

def softmax(logits):
    max_val = max(logits)
    exps = [math.exp(v - max_val) for v in logits]
    total = sum(exps)
    return [e / total for e in exps]

def rmsnorm(x):
    ms = sum(xi * xi for xi in x) / len(x)
    scale = (ms + 1e-5) ** -0.5
    return [xi * scale for xi in x]

def relu(x):
    return max(0, x)

# ============================================================
# Initialize all weights for one transformer block
# ============================================================
def make_matrix(rows, cols):
    return [[random.gauss(0, 0.3) for _ in range(cols)] for _ in range(rows)]

weights = {
    'attn_wq': make_matrix(n_embd, n_embd),
    'attn_wk': make_matrix(n_embd, n_embd),
    'attn_wv': make_matrix(n_embd, n_embd),
    'attn_wo': make_matrix(n_embd, n_embd),
    'mlp_fc1': make_matrix(4 * n_embd, n_embd),  # expand
    'mlp_fc2': make_matrix(n_embd, 4 * n_embd),   # compress
}

# ============================================================
# THE TRANSFORMER BLOCK
# ============================================================
def transformer_block(x, keys_cache, values_cache, pos):
    """
    Process one token through one transformer block.

    Args:
        x: input embedding vector [n_embd]
        keys_cache: list of previous key vectors
        values_cache: list of previous value vectors
        pos: position index (for printing)

    Returns:
        output vector [n_embd]
    """
    # ---- Part 1: Multi-Head Attention ----
    x_residual = x[:]  # save for residual connection

    # Pre-norm
    x_normed = rmsnorm(x)

    # Compute Q, K, V
    q = linear(x_normed, weights['attn_wq'])
    k = linear(x_normed, weights['attn_wk'])
    v = linear(x_normed, weights['attn_wv'])

    # Add current K, V to cache
    keys_cache.append(k)
    values_cache.append(v)

    # Multi-head attention
    x_attn = []
    for h in range(n_head):
        hs = h * head_dim
        q_h = q[hs:hs + head_dim]
        k_h = [ki[hs:hs + head_dim] for ki in keys_cache]
        v_h = [vi[hs:hs + head_dim] for vi in values_cache]

        # Attention scores
        attn_logits = [
            sum(q_h[j] * k_h[t][j] for j in range(head_dim)) / head_dim ** 0.5
            for t in range(len(k_h))
        ]
        attn_weights = softmax(attn_logits)

        # Weighted sum of values
        head_out = [
            sum(attn_weights[t] * v_h[t][j] for t in range(len(v_h)))
            for j in range(head_dim)
        ]
        x_attn.extend(head_out)

    # Output projection
    x = linear(x_attn, weights['attn_wo'])

    # Residual connection: ADD the input back
    x = [a + b for a, b in zip(x, x_residual)]

    # ---- Part 2: MLP ----
    x_residual = x[:]  # save for residual connection

    # Pre-norm
    x_normed = rmsnorm(x)

    # MLP: expand → activate → compress
    h = linear(x_normed, weights['mlp_fc1'])   # n_embd → 4*n_embd
    h = [relu(hi) for hi in h]                  # ReLU activation
    x_mlp = linear(h, weights['mlp_fc2'])       # 4*n_embd → n_embd

    # Residual connection: ADD the input back
    x = [a + b for a, b in zip(x_mlp, x_residual)]

    return x

# ============================================================
# DEMO: Process a sequence through the transformer block
# ============================================================
print("=" * 60)
print("Complete Transformer Block Demo")
print("=" * 60)

# Fake token embeddings
tokens = ["The", "cat", "sat", "on"]
X = [[random.gauss(0, 0.5) for _ in range(n_embd)] for _ in range(len(tokens))]

keys_cache = []
values_cache = []

print(f"\nProcessing {len(tokens)} tokens through one transformer block:\n")

for pos, token in enumerate(tokens):
    x_in = X[pos]
    x_out = transformer_block(x_in, keys_cache, values_cache, pos)

    # Compute how much the representation changed
    change = sum((a - b) ** 2 for a, b in zip(x_in, x_out)) ** 0.5

    print(f"  Token {pos} ('{token}'):")
    print(f"    Input:  [{', '.join(f'{v:+.3f}' for v in x_in[:4])}...]")
    print(f"    Output: [{', '.join(f'{v:+.3f}' for v in x_out[:4])}...]")
    print(f"    Change magnitude: {change:.4f}")
    print(f"    Attended to {pos + 1} token(s)")

print("""
=== Anatomy of a Transformer Block ===

  Input x
    │
    ├─── save as residual
    │
    ▼
  RMSNorm(x)
    │
    ▼
  Multi-Head Attention (look at previous tokens)
    │
    ▼
  + residual  ← Information highway! Original info preserved
    │
    ├─── save as residual
    │
    ▼
  RMSNorm(x)
    │
    ▼
  MLP: Linear(4x) → ReLU → Linear(1x)  (per-token processing)
    │
    ▼
  + residual  ← Information highway again
    │
    ▼
  Output x

=== Why Residual Connections? ===
Without them, information from early tokens would get lost.
The residual connection ensures the original signal always
has a direct path through the network. Think of it as:
  "Output = Original + What I Learned"

=== GPT = Stack of These Blocks ===
  - microGPT: 1 block  (tiny, for learning)
  - GPT-2 small: 12 blocks
  - GPT-2 XL: 48 blocks
  - GPT-3: 96 blocks

More blocks = more processing steps = more "thinking"
""")
