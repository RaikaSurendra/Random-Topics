# Chapter 11: Side-by-Side — microGPT vs minGPT

## The Same Algorithm, Two Implementations

Both implementations by Andrej Karpathy implement the **exact same GPT architecture**. The difference is in tooling and scale, not in ideas.

---

## Philosophy Comparison

| Aspect | microGPT | minGPT |
|---|---|---|
| **Goal** | "This is the complete algorithm. Everything else is just efficiency." | "Small, clean, interpretable and educational" |
| **Dependencies** | Zero (pure Python) | PyTorch |
| **Lines of code** | ~200 (single file) | ~850 (across 4 files) |
| **Speed** | Minutes per 200 steps | Seconds per 2000 steps |
| **GPU support** | No | Yes |
| **Batching** | 1 document at a time | Configurable batch size |
| **Best for** | Understanding the math | Building real projects |

---

## Architecture Mapping

```
microGPT                          minGPT
────────                          ──────
Value class                   →   torch.Tensor + autograd
state_dict['wte'][id]         →   nn.Embedding(vocab, n_embd)
state_dict['wpe'][id]         →   nn.Embedding(block_size, n_embd)
rmsnorm(x)                    →   nn.LayerNorm(n_embd)
linear(x, w) [manual loop]   →   nn.Linear(in, out) [GPU matmul]
xi.relu()                     →   NewGELU()
softmax(logits)               →   F.softmax(logits, dim=-1)
-probs[target].log()          →   F.cross_entropy(logits, targets)
Manual Adam loop              →   torch.optim.AdamW(...)
KV-cache (sequential)         →   Triangular mask (parallel)
```

---

## Run the Comparison

```bash
python comparison.py    # Detailed side-by-side code comparison
```
