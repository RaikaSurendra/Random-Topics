"""
Chapter 08: PyTorch Basics for Understanding minGPT

This introduces the PyTorch concepts you need to read minGPT's code.
Each section maps to something we built manually in earlier chapters.

Requires: pip install torch
"""

try:
    import torch
    import torch.nn as nn
    from torch.nn import functional as F
except ImportError:
    print("PyTorch not installed. Run: pip install torch")
    print("This chapter requires PyTorch.")
    exit(1)

# ============================================================
# 1. TENSORS — PyTorch's version of our Value class
# ============================================================
print("=" * 60)
print("1. Tensors — Multi-dimensional arrays with autograd")
print("=" * 60)

# Scalars, vectors, matrices, and higher
scalar = torch.tensor(3.14)
vector = torch.tensor([1.0, 2.0, 3.0])
matrix = torch.randn(3, 4)  # 3x4 matrix of random numbers

print(f"  Scalar: {scalar}  (shape: {scalar.shape})")
print(f"  Vector: {vector}  (shape: {vector.shape})")
print(f"  Matrix shape: {matrix.shape}")

# Matrix multiplication — what took nested loops in microGPT
A = torch.randn(2, 3)
B = torch.randn(3, 4)
C = A @ B  # matrix multiply — runs on GPU if available!
print(f"\n  Matrix multiply: ({list(A.shape)}) @ ({list(B.shape)}) → ({list(C.shape)})")
print(f"  This replaces our manual linear() function with nested loops!")

# ============================================================
# 2. AUTOGRAD — Automatic gradients (replaces our Value class)
# ============================================================
print("\n" + "=" * 60)
print("2. Autograd — Automatic differentiation")
print("=" * 60)

# Compare with Chapter 03's Value class
x = torch.tensor(2.0, requires_grad=True)
y = (2 * x + 3) ** 2  # y = (2*2+3)^2 = 25

y.backward()  # compute dy/dx

print(f"  x = {x.item():.1f}")
print(f"  y = (2x + 3)^2 = {y.item():.1f}")
print(f"  dy/dx = {x.grad.item():.1f}")
print(f"  Manual: 4*(2x+3) = 4*(2*2+3) = 20.0 ✓")

# ============================================================
# 3. nn.Linear — Replaces our manual linear() function
# ============================================================
print("\n" + "=" * 60)
print("3. nn.Linear — Built-in linear layer")
print("=" * 60)

# In microGPT: linear(x, w) with manual loops
# In minGPT:   nn.Linear does the same thing, but GPU-accelerated
layer = nn.Linear(3, 2)  # 3 inputs → 2 outputs

x = torch.tensor([1.0, 2.0, 3.0])
y = layer(x)  # applies y = Wx + b

print(f"  Input:  {x.tolist()} (3 values)")
print(f"  Output: [{', '.join(f'{v:.4f}' for v in y.tolist())}] (2 values)")
print(f"  Weight shape: {list(layer.weight.shape)}")
print(f"  Bias shape:   {list(layer.bias.shape)}")
print(f"  Total params: {layer.weight.numel() + layer.bias.numel()}")

# ============================================================
# 4. nn.Embedding — Replaces our embedding lookup table
# ============================================================
print("\n" + "=" * 60)
print("4. nn.Embedding — Token/position lookup table")
print("=" * 60)

# In microGPT: state_dict['wte'][token_id]
# In minGPT:   nn.Embedding(vocab_size, n_embd)
emb = nn.Embedding(10, 4)  # 10 tokens, each maps to 4 numbers

token_ids = torch.tensor([2, 5, 7])  # look up 3 tokens at once
vectors = emb(token_ids)

print(f"  Token IDs:  {token_ids.tolist()}")
print(f"  Output shape: {list(vectors.shape)}  (3 tokens × 4 dims)")
print(f"  Token 2 →  [{', '.join(f'{v:.3f}' for v in vectors[0].tolist())}]")
print(f"  Token 5 →  [{', '.join(f'{v:.3f}' for v in vectors[1].tolist())}]")
print(f"  Token 7 →  [{', '.join(f'{v:.3f}' for v in vectors[2].tolist())}]")

# ============================================================
# 5. SOFTMAX & CROSS-ENTROPY
# ============================================================
print("\n" + "=" * 60)
print("5. Softmax & Cross-Entropy — Built-in and optimized")
print("=" * 60)

logits = torch.tensor([2.0, 1.0, 0.5])
probs = F.softmax(logits, dim=-1)
print(f"  Logits:  {logits.tolist()}")
print(f"  Softmax: [{', '.join(f'{p:.4f}' for p in probs.tolist())}]")

# Cross-entropy loss (combines softmax + negative log likelihood)
logits_batch = torch.tensor([[2.0, 1.0, 0.5]])  # batch of 1
targets = torch.tensor([0])  # correct answer is token 0
loss = F.cross_entropy(logits_batch, targets)
print(f"  Cross-entropy loss: {loss.item():.4f}")
print(f"  Same as: -log(softmax[0]) = -log({probs[0].item():.4f}) = {-probs[0].log().item():.4f}")

# ============================================================
# 6. BATCHING — Processing multiple examples at once
# ============================================================
print("\n" + "=" * 60)
print("6. Batching — The big speed advantage")
print("=" * 60)

print("""
  microGPT processes ONE token at a time (sequential Python loops).
  minGPT processes MANY tokens at once using batch dimensions:

  microGPT: for each token in sequence: logits = gpt(token, pos)
  minGPT:   logits = model(all_tokens_at_once)  # one call!

  Shape conventions in minGPT:
    B = batch size (e.g., 32 documents at once)
    T = sequence length (e.g., 128 tokens)
    C = embedding dim (e.g., 768)

    Input:  (B, T)     — batch of token ID sequences
    After embedding: (B, T, C) — batch of embedding sequences
    Logits: (B, T, vocab_size) — predictions at every position
""")

# Demo: batch processing
B, T, C = 2, 4, 8  # 2 sequences, 4 tokens each, 8-dim embeddings
x = torch.randn(B, T, C)
linear_layer = nn.Linear(C, C)
y = linear_layer(x)  # applies to ALL positions in ALL sequences at once!
print(f"  Input shape:  {list(x.shape)} (B={B}, T={T}, C={C})")
print(f"  Output shape: {list(y.shape)} (same shape — processed all at once!)")

# ============================================================
# 7. nn.Module — Organizing models
# ============================================================
print("\n" + "=" * 60)
print("7. nn.Module — How minGPT organizes its model")
print("=" * 60)

class TinyModel(nn.Module):
    def __init__(self, n_in, n_hidden, n_out):
        super().__init__()
        self.layer1 = nn.Linear(n_in, n_hidden)
        self.layer2 = nn.Linear(n_hidden, n_out)

    def forward(self, x):
        x = F.relu(self.layer1(x))
        x = self.layer2(x)
        return x

model = TinyModel(3, 8, 2)
n_params = sum(p.numel() for p in model.parameters())
print(f"  Model: 3 → 8 → 2")
print(f"  Total parameters: {n_params}")
print(f"  Parameters are automatically tracked by nn.Module!")

# Forward and backward in one go
x = torch.randn(5, 3)  # batch of 5 inputs
y = model(x)
loss = y.sum()
loss.backward()

print(f"\n  Input shape:  {list(x.shape)}")
print(f"  Output shape: {list(y.shape)}")
print(f"  Gradients computed for all {n_params} parameters!")

print("""
=== Key Takeaway ===

PyTorch gives us the same building blocks as microGPT:
  Value class      → torch.Tensor with autograd
  manual linear()  → nn.Linear
  manual softmax() → F.softmax
  manual Adam      → torch.optim.AdamW

But MUCH faster because:
  - Operations run on GPU (or optimized CPU)
  - Batch processing: handle many sequences at once
  - No Python-level loops for math

Next: Chapter 09 — How minGPT uses all of this to build a full GPT model.
""")
