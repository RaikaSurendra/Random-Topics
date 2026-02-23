"""
Chapter 10: Text Generation — How GPT Creates Text

This script explains and demonstrates the text generation process,
mapping to both microGPT's and minGPT's generation code.

Requires: pip install torch
"""

try:
    import torch
    import torch.nn as nn
    from torch.nn import functional as F
    import math
except ImportError:
    print("PyTorch not installed. Run: pip install torch")
    exit(1)

torch.manual_seed(42)

# ============================================================
# TEXT GENERATION EXPLAINED
# ============================================================
print("=" * 60)
print("How GPT Generates Text")
print("=" * 60)
print("""
Generation is an iterative process:

  1. Start with a prompt (or just a start token)
  2. Feed it through the model → get logits for ALL vocab tokens
  3. Take only the LAST position's logits (prediction for next token)
  4. Apply temperature and optional top-k filtering
  5. Convert to probabilities (softmax)
  6. Sample a token from the distribution
  7. Append it to the sequence
  8. Repeat from step 2
""")

# ============================================================
# STEP-BY-STEP GENERATION DEMO (no model, just the mechanics)
# ============================================================
print("=" * 60)
print("Step-by-Step: Temperature & Sampling")
print("=" * 60)

vocab = ['a', 'b', 'c', 'd', 'e']
logits = torch.tensor([2.0, 5.0, 1.0, 0.5, 0.1])

print(f"\nRaw logits: {logits.tolist()}")
print(f"Vocab: {vocab}")

# Temperature effect
print("\n--- Temperature Effect ---")
for temp in [0.1, 0.5, 1.0, 1.5, 3.0]:
    scaled = logits / temp
    probs = F.softmax(scaled, dim=-1)
    print(f"  temp={temp:.1f}: [{', '.join(f'{p:.3f}' for p in probs.tolist())}]", end="")
    if temp < 0.5:
        print("  ← very peaked (almost deterministic)")
    elif temp < 1.0:
        print("  ← confident")
    elif temp == 1.0:
        print("  ← normal")
    else:
        print("  ← more random")

# Top-k filtering
print("\n--- Top-k Filtering ---")
print("  Only consider the k most likely tokens, set rest to -inf")
for k in [1, 2, 3, 5]:
    filtered = logits.clone()
    if k < len(logits):
        v, _ = torch.topk(logits, k)
        filtered[filtered < v[-1]] = float('-inf')
    probs = F.softmax(filtered, dim=-1)
    top_tokens = [vocab[i] for i in range(len(vocab)) if probs[i] > 0]
    print(f"  top_k={k}: [{', '.join(f'{p:.3f}' for p in probs.tolist())}] "
          f"candidates: {top_tokens}")

# Sampling vs Greedy
print("\n--- Sampling vs Greedy ---")
probs = F.softmax(logits, dim=-1)
print(f"  Probabilities: [{', '.join(f'{p:.3f}' for p in probs.tolist())}]")

# Greedy: always pick the most likely
_, greedy_idx = torch.topk(probs, k=1)
print(f"  Greedy (argmax):  always picks '{vocab[greedy_idx.item()]}' (highest prob)")

# Sampling: randomly pick, weighted by probability
print(f"  Sampling (10 tries): ", end="")
samples = []
for _ in range(10):
    idx = torch.multinomial(probs, num_samples=1)
    samples.append(vocab[idx.item()])
print(' '.join(samples))
print(f"  ('{vocab[1]}' appears most often because it has highest probability)")

# ============================================================
# THE GENERATE FUNCTION (from minGPT)
# ============================================================
print("\n" + "=" * 60)
print("The Generate Function — Code Comparison")
print("=" * 60)

print("""
microGPT (lines 189-200):                 minGPT (model.py lines 283-310):
─────────────────────────                  ──────────────────────────────────
token_id = BOS                             idx = starting_tokens
for pos_id in range(block_size):           for _ in range(max_new_tokens):
    logits = gpt(token_id, pos_id, ...)        idx_cond = idx[:, -block_size:]
    probs = softmax([l/temp for l in ..])      logits, _ = self(idx_cond)
    token_id = random.choices(...)             logits = logits[:, -1, :] / temp
    if token_id == BOS: break                  probs = F.softmax(logits, dim=-1)
    sample.append(uchars[token_id])            idx_next = torch.multinomial(...)
                                               idx = torch.cat((idx, idx_next), 1)

Same algorithm:
  1. Get logits from model
  2. Scale by temperature
  3. Softmax → probabilities
  4. Sample next token
  5. Append and repeat

Key difference: minGPT processes in batches and handles
sequences longer than block_size by cropping.
""")

# ============================================================
# HOW CHARGPT GENERATES SHAKESPEARE
# ============================================================
print("=" * 60)
print("Real Application: CharGPT on Shakespeare")
print("=" * 60)

print("""
The minGPT project 'chargpt' trains on Shakespeare's text:

  Source: ../../minGPT/projects/chargpt/chargpt.py

  1. Load text file (e.g., all of Shakespeare)
  2. Build character vocabulary ({'a':0, 'b':1, ..., 'z':25, ' ':26, ...})
  3. Create a CharDataset:
     - Each training example is a random 128-character chunk
     - Input: characters [0:127], Target: characters [1:128]
  4. Train gpt-mini (6 layers, 192 dim) for thousands of iterations
  5. Every 500 steps, generate a sample starting from "O God, O God!"

  After training, it produces text like:
  "O God, O God! what shall I say to thee?
   That I have lost my father, and my friend,
   And all my mother's sons, and all my kin..."

  It learned:
  - English spelling and grammar
  - Shakespeare's vocabulary and style
  - Poetic meter (roughly)
  - Character names and dialogue patterns

  All from raw character sequences — no rules, no grammar engine,
  just pattern matching at scale.
""")

# ============================================================
# THE COMPLETE GENERATION PIPELINE
# ============================================================
print("=" * 60)
print("Summary: The Complete Generation Pipeline")
print("=" * 60)

print("""
  ┌─────────────────────────────────────────────────┐
  │                  GENERATION                      │
  │                                                  │
  │  Prompt: "Hello"                                 │
  │       ↓                                          │
  │  Tokenize: [H, e, l, l, o] → [7, 4, 11, 11, 14]│
  │       ↓                                          │
  │  ┌─── LOOP (repeat for each new token) ─────┐   │
  │  │                                           │   │
  │  │  Feed tokens through GPT model            │   │
  │  │       ↓                                   │   │
  │  │  Get logits at last position               │   │
  │  │       ↓                                   │   │
  │  │  Apply temperature (divide by T)          │   │
  │  │       ↓                                   │   │
  │  │  Optional: top-k filtering                │   │
  │  │       ↓                                   │   │
  │  │  Softmax → probabilities                  │   │
  │  │       ↓                                   │   │
  │  │  Sample next token                        │   │
  │  │       ↓                                   │   │
  │  │  Append to sequence                       │   │
  │  │                                           │   │
  │  └───────────────────────────────────────────┘   │
  │       ↓                                          │
  │  Decode tokens → text: "Hello world, I am..."   │
  │                                                  │
  └─────────────────────────────────────────────────┘

  Controls:
    temperature < 1.0 → more focused, repetitive
    temperature > 1.0 → more creative, random
    top_k = small     → only consider top few options
    do_sample = False → always pick best (greedy/deterministic)
""")
