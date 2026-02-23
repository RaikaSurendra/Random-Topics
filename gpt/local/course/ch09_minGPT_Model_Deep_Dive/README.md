# Chapter 09: minGPT Model — Deep Dive

## Overview

minGPT's model is defined in `mingpt/model.py` (~311 lines). It implements the **exact same architecture** as microGPT but using PyTorch, making it fast enough to train on real tasks.

**Source:** `../../minGPT/mingpt/model.py`

---

## The Four Classes

### 1. `NewGELU` (line 21-27)
Activation function — smoother alternative to ReLU.
- microGPT uses `ReLU(x) = max(0, x)`
- minGPT uses `GELU(x) ≈ 0.5 * x * (1 + tanh(...))`

### 2. `CausalSelfAttention` (line 29-71)
Multi-head attention with causal masking.
- Same math as microGPT's attention, but batched
- Uses a pre-computed triangular mask for causal attention
- All Q/K/V computed in one matrix multiply (efficiency trick)

### 3. `Block` (line 73-93)
One transformer block = Attention + MLP + residual connections.
- Identical structure to microGPT's transformer block
- Uses LayerNorm instead of RMSNorm

### 4. `GPT` (line 95-311)
The full model: embeddings + stacked blocks + output head.
- Supports multiple model sizes (gpt-nano to gpt2-xl)
- Can load pretrained GPT-2 weights from HuggingFace
- Includes generation (inference) logic

---

## Model Size Configurations

```python
'gpt-nano':    n_layer=3,  n_head=3,  n_embd=48    # Tiny, for demos
'gpt-micro':   n_layer=4,  n_head=4,  n_embd=128   # Small
'gpt-mini':    n_layer=6,  n_head=6,  n_embd=192   # Medium-small
'gpt2':        n_layer=12, n_head=12, n_embd=768   # 124M params (real GPT-2)
'gpt2-xl':     n_layer=48, n_head=25, n_embd=1600  # 1.5B params
```

---

## The Forward Pass (simplified)

```
Input token IDs: (batch_size, seq_len)
        ↓
Token Embedding + Position Embedding
        ↓
Dropout
        ↓
Block 0: LayerNorm → Attention → (+) → LayerNorm → MLP → (+)
Block 1: LayerNorm → Attention → (+) → LayerNorm → MLP → (+)
  ...
Block N: LayerNorm → Attention → (+) → LayerNorm → MLP → (+)
        ↓
Final LayerNorm
        ↓
Linear head → logits (batch_size, seq_len, vocab_size)
        ↓
Cross-entropy loss (if targets provided)
```

---

## Run the Examples

```bash
python model_walkthrough.py    # Annotated walkthrough of minGPT's model.py
python model_sizes.py          # Compare different GPT configurations
```

**Requires:** `pip install torch`
