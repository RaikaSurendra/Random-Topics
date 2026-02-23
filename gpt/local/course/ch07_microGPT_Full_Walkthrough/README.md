# Chapter 07: microGPT — Full Walkthrough

## What is microGPT?

microGPT is a **complete GPT implementation in 200 lines of pure Python**. No PyTorch, no TensorFlow, no libraries (except `math`, `random`, `os`). It builds everything from scratch:

- Autograd engine (Chapter 03)
- Neural network layers (Chapter 04)
- Multi-head attention (Chapter 05)
- Training loop with Adam optimizer (Chapter 06)

It trains on a dataset of baby names and learns to generate new, plausible-sounding names.

**Source:** `../../8627fe009c40f57531cb18360106ce95/microgpt.py`

---

## The 7 Sections of microGPT

| Lines | Section | What It Does |
|---|---|---|
| 1-21 | Dataset | Downloads names, shuffles them |
| 23-27 | Tokenizer | Character-level: each letter → integer |
| 29-73 | Autograd | The `Value` class with forward/backward |
| 74-90 | Parameters | Initialize all weight matrices |
| 92-144 | Model | The GPT architecture (attention + MLP) |
| 146-184 | Training | Forward → loss → backward → Adam update |
| 186-200 | Inference | Generate new names from the trained model |

---

## Architecture Summary

```
microGPT Config:
  n_layer = 1       (1 transformer block)
  n_embd = 16       (16-dimensional embeddings)
  block_size = 16   (max sequence length)
  n_head = 4        (4 attention heads)
  head_dim = 4      (16 / 4 = 4 dims per head)
  vocab_size = 27   (26 letters + BOS token)
```

Total parameters: ~5,000 (vs GPT-2's 124 million!)

---

## Run the Annotated Version

```bash
python microgpt_annotated.py   # The full microGPT with detailed comments
```

**Warning:** This takes several minutes to run because it's pure Python (no GPU acceleration). That's the tradeoff for zero dependencies!

---

## Key Differences from "Real" GPT-2

| Feature | microGPT | GPT-2 |
|---|---|---|
| Dependencies | None | PyTorch |
| Autograd | Custom `Value` class | PyTorch autograd |
| Normalization | RMSNorm | LayerNorm |
| Activation | ReLU | GELU |
| Biases | None | Yes |
| Layers | 1 | 12-48 |
| Parameters | ~5,000 | 124M-1.5B |
| Training | Single document at a time | Batched |
| Tokenizer | Character-level | BPE (50K vocab) |
