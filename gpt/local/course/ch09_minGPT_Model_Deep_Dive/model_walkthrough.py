"""
Chapter 09: minGPT Model Walkthrough

This script builds a minGPT model step-by-step, explaining
every component and how it maps to the microGPT concepts
from Chapter 07.

Source: ../../minGPT/mingpt/model.py

Requires: pip install torch
"""

try:
    import math
    import torch
    import torch.nn as nn
    from torch.nn import functional as F
except ImportError:
    print("PyTorch not installed. Run: pip install torch")
    exit(1)

torch.manual_seed(42)

# ============================================================
# COMPONENT 1: NewGELU Activation
# ============================================================
# In microGPT:  x.relu()  →  max(0, x)
# In minGPT:    NewGELU   →  smoother, slightly better for training

print("=" * 60)
print("Component 1: NewGELU Activation")
print("=" * 60)

class NewGELU(nn.Module):
    """
    GELU = Gaussian Error Linear Unit
    Used in GPT-2 instead of ReLU.
    Almost identical to ReLU for positive values,
    but allows small negative values through.
    """
    def forward(self, x):
        return 0.5 * x * (1.0 + torch.tanh(
            math.sqrt(2.0 / math.pi) * (x + 0.044715 * torch.pow(x, 3.0))
        ))

gelu = NewGELU()
test_vals = torch.tensor([-2.0, -1.0, -0.5, 0.0, 0.5, 1.0, 2.0])
print(f"  Input:  {test_vals.tolist()}")
print(f"  GELU:   [{', '.join(f'{v:.3f}' for v in gelu(test_vals).tolist())}]")
print(f"  ReLU:   [{', '.join(f'{v:.3f}' for v in F.relu(test_vals).tolist())}]")
print("  Notice: GELU is smoother — small negatives get small negative outputs")

# ============================================================
# COMPONENT 2: CausalSelfAttention
# ============================================================
print("\n" + "=" * 60)
print("Component 2: CausalSelfAttention")
print("=" * 60)

class CausalSelfAttention(nn.Module):
    """
    Multi-head masked self-attention.
    
    microGPT equivalent: the attention section in gpt() function (lines 114-133)
    
    Key differences from microGPT:
    - Processes ALL positions at once (batched), not one-by-one
    - Q, K, V computed in ONE matrix multiply (c_attn), not three
    - Uses a triangular mask instead of KV-cache for causal masking
    - Includes dropout for regularization
    """
    def __init__(self, n_embd, n_head, block_size):
        super().__init__()
        assert n_embd % n_head == 0
        # Combined Q, K, V projection (efficiency: one matmul instead of three)
        self.c_attn = nn.Linear(n_embd, 3 * n_embd)
        # Output projection
        self.c_proj = nn.Linear(n_embd, n_embd)
        # Causal mask: lower triangular matrix
        self.register_buffer("bias",
            torch.tril(torch.ones(block_size, block_size)).view(1, 1, block_size, block_size))
        self.n_head = n_head
        self.n_embd = n_embd

    def forward(self, x):
        B, T, C = x.size()  # batch, sequence length, embedding dim

        # Compute Q, K, V all at once, then split
        q, k, v = self.c_attn(x).split(self.n_embd, dim=2)

        # Reshape for multi-head: (B, T, C) → (B, n_head, T, head_dim)
        k = k.view(B, T, self.n_head, C // self.n_head).transpose(1, 2)
        q = q.view(B, T, self.n_head, C // self.n_head).transpose(1, 2)
        v = v.view(B, T, self.n_head, C // self.n_head).transpose(1, 2)

        # Attention scores: (B, nh, T, hs) × (B, nh, hs, T) → (B, nh, T, T)
        att = (q @ k.transpose(-2, -1)) * (1.0 / math.sqrt(k.size(-1)))

        # Causal mask: set future positions to -inf
        att = att.masked_fill(self.bias[:, :, :T, :T] == 0, float('-inf'))

        # Softmax → attention weights
        att = F.softmax(att, dim=-1)

        # Weighted sum of values
        y = att @ v  # (B, nh, T, T) × (B, nh, T, hs) → (B, nh, T, hs)

        # Concatenate heads: (B, nh, T, hs) → (B, T, C)
        y = y.transpose(1, 2).contiguous().view(B, T, C)

        # Output projection
        y = self.c_proj(y)
        return y

# Demo
n_embd, n_head, block_size = 16, 4, 8
attn = CausalSelfAttention(n_embd, n_head, block_size)
x = torch.randn(2, 5, n_embd)  # batch=2, seq_len=5
y = attn(x)
print(f"  Config: n_embd={n_embd}, n_head={n_head}")
print(f"  Input:  {list(x.shape)} (batch=2, tokens=5, dim={n_embd})")
print(f"  Output: {list(y.shape)} (same shape — attention transforms, doesn't resize)")
print(f"  Params: {sum(p.numel() for p in attn.parameters())}")

print(f"\n  Causal mask (first 5×5):")
mask = attn.bias[0, 0, :5, :5]
for i in range(5):
    row = ['✓' if mask[i, j] == 1 else '✗' for j in range(5)]
    print(f"    pos {i}: {' '.join(row)}  (can attend to {int(mask[i].sum())} positions)")

# ============================================================
# COMPONENT 3: Block (Transformer Block)
# ============================================================
print("\n" + "=" * 60)
print("Component 3: Transformer Block")
print("=" * 60)

class Block(nn.Module):
    """
    One transformer block: Attention + MLP with residual connections.
    
    microGPT equivalent: one iteration of the `for li in range(n_layer)` loop
    
    Structure:
        x → LayerNorm → Attention → (+x) → LayerNorm → MLP → (+x) → output
    """
    def __init__(self, n_embd, n_head, block_size):
        super().__init__()
        self.ln_1 = nn.LayerNorm(n_embd)        # Pre-attention norm
        self.attn = CausalSelfAttention(n_embd, n_head, block_size)
        self.ln_2 = nn.LayerNorm(n_embd)        # Pre-MLP norm
        self.mlp = nn.ModuleDict(dict(
            c_fc   = nn.Linear(n_embd, 4 * n_embd),   # Expand
            c_proj = nn.Linear(4 * n_embd, n_embd),    # Compress
            act    = NewGELU(),
        ))

    def forward(self, x):
        # Attention with residual connection
        x = x + self.attn(self.ln_1(x))
        # MLP with residual connection
        m = self.mlp
        x = x + m.c_proj(m.act(m.c_fc(self.ln_2(x))))
        return x

block = Block(n_embd, n_head, block_size)
x = torch.randn(2, 5, n_embd)
y = block(x)
n_params = sum(p.numel() for p in block.parameters())
print(f"  Input:  {list(x.shape)}")
print(f"  Output: {list(y.shape)}")
print(f"  Block params: {n_params}")
print(f"  Components: LayerNorm + Attention + LayerNorm + MLP(expand→GELU→compress)")

# ============================================================
# COMPONENT 4: Full GPT Model
# ============================================================
print("\n" + "=" * 60)
print("Component 4: Full GPT Model")
print("=" * 60)

class MiniGPT(nn.Module):
    """
    Simplified version of minGPT's GPT class for illustration.
    Same architecture, just without the config system and pretrained loading.
    """
    def __init__(self, vocab_size, block_size, n_layer, n_head, n_embd):
        super().__init__()
        self.block_size = block_size

        self.transformer = nn.ModuleDict(dict(
            wte  = nn.Embedding(vocab_size, n_embd),          # Token embeddings
            wpe  = nn.Embedding(block_size, n_embd),          # Position embeddings
            h    = nn.ModuleList([Block(n_embd, n_head, block_size)
                                  for _ in range(n_layer)]),  # Transformer blocks
            ln_f = nn.LayerNorm(n_embd),                      # Final norm
        ))
        self.lm_head = nn.Linear(n_embd, vocab_size, bias=False)  # Output projection

    def forward(self, idx, targets=None):
        B, T = idx.size()
        pos = torch.arange(0, T, dtype=torch.long, device=idx.device)

        # Embeddings
        tok_emb = self.transformer.wte(idx)    # (B, T, n_embd)
        pos_emb = self.transformer.wpe(pos)    # (T, n_embd) → broadcasts to (B, T, n_embd)
        x = tok_emb + pos_emb

        # Transformer blocks
        for block in self.transformer.h:
            x = block(x)

        # Final norm + output projection
        x = self.transformer.ln_f(x)
        logits = self.lm_head(x)               # (B, T, vocab_size)

        # Compute loss if targets given
        loss = None
        if targets is not None:
            loss = F.cross_entropy(logits.view(-1, logits.size(-1)), targets.view(-1))

        return logits, loss

# Create a tiny GPT
model = MiniGPT(
    vocab_size=27,    # 26 letters + BOS (like microGPT)
    block_size=16,    # max sequence length
    n_layer=3,        # 3 transformer blocks
    n_head=3,         # 3 attention heads
    n_embd=48,        # 48-dim embeddings
)

n_params = sum(p.numel() for p in model.parameters())
print(f"  Model: vocab=27, block_size=16, layers=3, heads=3, embd=48")
print(f"  Total parameters: {n_params:,}")

# Test forward pass
idx = torch.randint(0, 27, (2, 10))     # batch of 2 sequences, 10 tokens each
targets = torch.randint(0, 27, (2, 10))  # target tokens

logits, loss = model(idx, targets)
print(f"\n  Forward pass:")
print(f"    Input shape:  {list(idx.shape)} (2 sequences × 10 tokens)")
print(f"    Logits shape: {list(logits.shape)} (2 × 10 × 27 vocab)")
print(f"    Loss: {loss.item():.4f}")
print(f"    (Random init → loss ≈ -log(1/27) = {-math.log(1/27):.4f})")

print("""
=== How This Maps to microGPT ===

minGPT Component        →  microGPT Equivalent
─────────────────────────────────────────────────
nn.Embedding(wte)       →  state_dict['wte'][token_id]
nn.Embedding(wpe)       →  state_dict['wpe'][pos_id]
CausalSelfAttention     →  The attention loop in gpt()
Block                   →  One iteration of for li in range(n_layer)
nn.LayerNorm            →  rmsnorm()
NewGELU                 →  .relu()
nn.Linear(lm_head)      →  linear(x, state_dict['lm_head'])
F.cross_entropy         →  -probs[target_id].log()

Same architecture. Same math. Different scale and speed.
""")
