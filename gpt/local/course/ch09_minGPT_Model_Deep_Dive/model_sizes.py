"""
Chapter 09: GPT Model Sizes — From Nano to GPT-2

Shows how the same architecture scales from tiny to huge
by changing just three numbers: n_layer, n_head, n_embd.

Requires: pip install torch
"""

try:
    import torch
    import torch.nn as nn
    import math
except ImportError:
    print("PyTorch not installed. Run: pip install torch")
    exit(1)

# ============================================================
# Calculate parameter count for a GPT model
# ============================================================
def count_gpt_params(vocab_size, block_size, n_layer, n_head, n_embd):
    """Count parameters in a GPT model without building it."""
    params = {}

    # Token embeddings
    params['wte'] = vocab_size * n_embd

    # Position embeddings
    params['wpe'] = block_size * n_embd

    # Per-layer parameters
    per_layer = {}
    # Attention: c_attn (Q,K,V combined) + c_proj (output)
    per_layer['c_attn_weight'] = n_embd * (3 * n_embd)
    per_layer['c_attn_bias'] = 3 * n_embd
    per_layer['c_proj_weight'] = n_embd * n_embd
    per_layer['c_proj_bias'] = n_embd
    # MLP: c_fc (expand) + c_proj (compress)
    per_layer['mlp_fc_weight'] = n_embd * (4 * n_embd)
    per_layer['mlp_fc_bias'] = 4 * n_embd
    per_layer['mlp_proj_weight'] = (4 * n_embd) * n_embd
    per_layer['mlp_proj_bias'] = n_embd
    # LayerNorms (2 per layer)
    per_layer['ln1'] = 2 * n_embd  # weight + bias
    per_layer['ln2'] = 2 * n_embd

    layer_total = sum(per_layer.values())
    params['all_layers'] = layer_total * n_layer

    # Final LayerNorm
    params['ln_f'] = 2 * n_embd

    # Output head (often shares weights with wte, but let's count separately)
    params['lm_head'] = n_embd * vocab_size

    total = sum(params.values())
    return total, params, per_layer, layer_total


# ============================================================
# Compare model sizes
# ============================================================
print("=" * 70)
print("GPT Model Size Comparison")
print("=" * 70)

configs = [
    # name, vocab_size, block_size, n_layer, n_head, n_embd
    ("microGPT",    27,    16,   1,  4,   16),
    ("gpt-nano",    50257, 1024, 3,  3,   48),
    ("gpt-micro",   50257, 1024, 4,  4,   128),
    ("gpt-mini",    50257, 1024, 6,  6,   192),
    ("GPT-2",       50257, 1024, 12, 12,  768),
    ("GPT-2 medium",50257, 1024, 24, 16,  1024),
    ("GPT-2 large", 50257, 1024, 36, 20,  1280),
    ("GPT-2 XL",    50257, 1024, 48, 25,  1600),
]

print(f"\n{'Model':<16} {'Layers':>6} {'Heads':>6} {'Embd':>6} {'Params':>14} {'Relative':>10}")
print("-" * 70)

base_params = None
for name, vocab, block, n_layer, n_head, n_embd in configs:
    total, _, _, _ = count_gpt_params(vocab, block, n_layer, n_head, n_embd)
    if base_params is None:
        base_params = total
    relative = total / base_params
    
    if total < 1_000_000:
        param_str = f"{total:,}"
    else:
        param_str = f"{total/1e6:.1f}M"
    
    print(f"{name:<16} {n_layer:>6} {n_head:>6} {n_embd:>6} {param_str:>14} {relative:>9.0f}x")

# ============================================================
# Detailed breakdown of one model
# ============================================================
print("\n" + "=" * 70)
print("Detailed Breakdown: gpt-mini (n_layer=6, n_head=6, n_embd=192)")
print("=" * 70)

total, params, per_layer, layer_total = count_gpt_params(50257, 1024, 6, 6, 192)

print(f"\n  Token embeddings (wte):    {params['wte']:>10,}  ({params['wte']/total*100:.1f}%)")
print(f"  Position embeddings (wpe): {params['wpe']:>10,}  ({params['wpe']/total*100:.1f}%)")
print(f"  All transformer layers:    {params['all_layers']:>10,}  ({params['all_layers']/total*100:.1f}%)")
print(f"  Final LayerNorm:           {params['ln_f']:>10,}  ({params['ln_f']/total*100:.1f}%)")
print(f"  Output head (lm_head):     {params['lm_head']:>10,}  ({params['lm_head']/total*100:.1f}%)")
print(f"  {'─'*40}")
print(f"  TOTAL:                     {total:>10,}")

print(f"\n  Per-layer breakdown ({layer_total:,} params per layer × 6 layers):")
for name, count in per_layer.items():
    print(f"    {name:<20} {count:>8,}  ({count/layer_total*100:.1f}%)")

# ============================================================
# What each parameter "does"
# ============================================================
print("""
=== What Each Component Stores ===

Token Embeddings (wte):
  A lookup table: token ID → vector. Captures "meaning" of each token.
  Similar tokens (e.g., "cat", "dog") end up with similar vectors.

Position Embeddings (wpe):
  A lookup table: position → vector. Captures "where in the sequence".
  Without this, the model can't tell position 1 from position 100.

Attention Weights (c_attn, c_proj):
  Control HOW tokens communicate. Q/K weights determine what each
  token "looks for" and "advertises". V/output weights control what
  information flows between tokens.

MLP Weights (mlp_fc, mlp_proj):
  Per-token processing. Each token's representation gets "refined"
  through an expand→activate→compress pipeline. This is where a lot
  of the model's "knowledge" is stored (facts, patterns, etc).

=== The Scaling Insight ===

Notice: most parameters are in the transformer layers (not embeddings).
Doubling n_embd roughly QUADRUPLES the parameter count (because weight
matrices are n_embd × n_embd). This is why bigger models are so expensive.
""")
