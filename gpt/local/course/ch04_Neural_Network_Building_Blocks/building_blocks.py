"""
Chapter 04: Neural Network Building Blocks — Hands-On

Every building block used in GPT, implemented from scratch.
These are the SAME functions used in microGPT (microgpt.py).
"""

import math
import random

random.seed(42)

# ============================================================
# BLOCK 1: LINEAR LAYER
# ============================================================
# A linear layer is just matrix-vector multiplication.
# Each output = weighted sum of all inputs.

def linear(x, w):
    """
    x: input vector of length n_in
    w: weight matrix of shape [n_out][n_in]
    returns: output vector of length n_out
    """
    return [sum(wi * xi for wi, xi in zip(row, x)) for row in w]

print("=" * 50)
print("BLOCK 1: Linear Layer")
print("=" * 50)

# Input: a vector of 3 numbers
x = [1.0, 2.0, 3.0]

# Weights: a 2x3 matrix (maps 3 inputs → 2 outputs)
w = [
    [0.1, 0.2, 0.3],  # weights for output 0
    [0.4, 0.5, 0.6],  # weights for output 1
]

y = linear(x, w)
print(f"Input:   {x}  (3 values)")
print(f"Weights: {w}")
print(f"Output:  {y}  (2 values)")
print(f"  y[0] = 0.1*1 + 0.2*2 + 0.3*3 = {0.1*1 + 0.2*2 + 0.3*3}")
print(f"  y[1] = 0.4*1 + 0.5*2 + 0.6*3 = {0.4*1 + 0.5*2 + 0.6*3}")

# ============================================================
# BLOCK 2: ACTIVATION FUNCTIONS
# ============================================================
print("\n" + "=" * 50)
print("BLOCK 2: Activation Functions")
print("=" * 50)

def relu(x):
    """ReLU: zero out negative values. Used in microGPT."""
    return max(0, x)

print("\nReLU (used in microGPT):")
for val in [-2, -1, 0, 1, 2, 3]:
    print(f"  relu({val:2d}) = {relu(val)}")

print("\nWhy ReLU? Without it, stacking linear layers = one big linear layer.")
print("ReLU adds 'kinks' so the network can model non-linear patterns.")

# GELU (used in minGPT/GPT-2) — smoother version of ReLU
def gelu(x):
    """GELU: Gaussian Error Linear Unit. Used in GPT-2/minGPT."""
    return 0.5 * x * (1.0 + math.tanh(math.sqrt(2.0 / math.pi) * (x + 0.044715 * x**3)))

print("\nGELU (used in minGPT/GPT-2):")
for val in [-2, -1, 0, 1, 2, 3]:
    print(f"  gelu({val:2d}) = {gelu(val):.4f}")

print("\nGELU is smoother than ReLU — slightly negative inputs get small negative outputs")
print("instead of being hard-zeroed. This helps training in practice.")

# ============================================================
# BLOCK 3: SOFTMAX
# ============================================================
print("\n" + "=" * 50)
print("BLOCK 3: Softmax")
print("=" * 50)

def softmax(logits):
    """Convert raw scores to probabilities."""
    max_val = max(logits)
    exps = [math.exp(v - max_val) for v in logits]  # subtract max for numerical stability
    total = sum(exps)
    return [e / total for e in exps]

scores = [2.0, 1.0, 0.5]
probs = softmax(scores)
print(f"\nRaw scores (logits): {scores}")
print(f"After softmax:       [{', '.join(f'{p:.4f}' for p in probs)}]")
print(f"Sum of probabilities: {sum(probs):.4f}")
print(f"\nHigher score → higher probability. All positive, sum to 1.")

# Show the effect of "temperature"
print("\nTemperature effect (used during text generation):")
for temp in [0.5, 1.0, 2.0]:
    scaled = [s / temp for s in scores]
    p = softmax(scaled)
    print(f"  temp={temp}: [{', '.join(f'{x:.3f}' for x in p)}]  ", end="")
    if temp < 1:
        print("← more confident (peaked)")
    elif temp > 1:
        print("← more random (flat)")
    else:
        print("← normal")

# ============================================================
# BLOCK 4: RMSNORM / LAYERNORM
# ============================================================
print("\n" + "=" * 50)
print("BLOCK 4: Normalization")
print("=" * 50)

def rmsnorm(x):
    """RMSNorm: scale so root-mean-square = 1. Used in microGPT."""
    ms = sum(xi * xi for xi in x) / len(x)
    scale = (ms + 1e-5) ** -0.5
    return [xi * scale for xi in x]

def layernorm(x):
    """LayerNorm: shift to mean=0, scale to std=1. Used in minGPT."""
    mean = sum(x) / len(x)
    variance = sum((xi - mean) ** 2 for xi in x) / len(x)
    scale = (variance + 1e-5) ** -0.5
    return [(xi - mean) * scale for xi in x]

x = [10.0, 20.0, 30.0, 40.0]
print(f"\nOriginal:   {x}")
print(f"RMSNorm:    [{', '.join(f'{v:.4f}' for v in rmsnorm(x))}]")
print(f"LayerNorm:  [{', '.join(f'{v:.4f}' for v in layernorm(x))}]")

print("\nWhy normalize? Keeps values in a reasonable range.")
print("Without it, values can grow huge or tiny, making training unstable.")

# ============================================================
# BLOCK 5: EMBEDDING
# ============================================================
print("\n" + "=" * 50)
print("BLOCK 5: Embedding (Lookup Table)")
print("=" * 50)

# An embedding is just a table of vectors, one per token
n_embd = 4  # each token becomes a vector of 4 numbers
vocab_size = 5  # we have 5 possible tokens

# Initialize randomly (these get learned during training)
embedding_table = [[random.gauss(0, 0.1) for _ in range(n_embd)]
                   for _ in range(vocab_size)]

print(f"\nEmbedding table ({vocab_size} tokens × {n_embd} dimensions):")
for i, row in enumerate(embedding_table):
    print(f"  Token {i} → [{', '.join(f'{v:+.3f}' for v in row)}]")

token_id = 3
print(f"\nLooking up token {token_id}:")
print(f"  → [{', '.join(f'{v:+.3f}' for v in embedding_table[token_id])}]")
print("\nEmbeddings are LEARNED — after training, similar tokens")
print("end up with similar vectors (e.g., 'cat' close to 'dog').")

print("""
=== Summary ===
These 5 blocks are ALL you need to build GPT:
  1. Linear    → combine information (weighted sums)
  2. Activation → add non-linearity (ReLU or GELU)
  3. Softmax   → get probabilities for next token
  4. Norm      → keep numbers stable (RMSNorm or LayerNorm)
  5. Embedding → turn token IDs into vectors

Next chapter: Attention — the special sauce of Transformers!
""")
