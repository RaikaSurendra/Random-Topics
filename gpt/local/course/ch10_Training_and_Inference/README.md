# Chapter 10: Training & Inference with minGPT

## Overview

This chapter covers the two remaining pieces of minGPT:
1. **Trainer** (`mingpt/trainer.py`) — The training loop
2. **Generation** (`GPT.generate()`) — How the model creates new text

Plus two real demo projects:
- **Sorting demo** — Teaching GPT to sort numbers
- **CharGPT** — Training on Shakespeare to generate new plays

---

## The Trainer (mingpt/trainer.py)

minGPT's Trainer is ~110 lines of clean PyTorch boilerplate:

```python
while True:
    x, y = next(data_iter)              # 1. Get batch of data
    logits, loss = model(x, y)          # 2. Forward pass
    model.zero_grad()                    # 3. Reset gradients
    loss.backward()                      # 4. Backward pass (autograd)
    clip_grad_norm_(model.parameters())  # 5. Prevent gradient explosion
    optimizer.step()                      # 6. Adam update
```

Compare to microGPT's training loop — identical structure!

---

## The Optimizer Setup

minGPT is careful about **weight decay** (regularization):
- **Decay** weights in Linear layers (prevents overfitting)
- **Don't decay** biases, LayerNorm weights, embeddings

This is handled in `GPT.configure_optimizers()`.

---

## Text Generation (Inference)

`GPT.generate()` works like microGPT's inference section:

```python
for each new token to generate:
    1. Forward pass → get logits for last position
    2. Apply temperature (divide logits by temperature)
    3. Optionally apply top-k filtering
    4. Softmax → probabilities
    5. Sample from distribution (or take argmax)
    6. Append new token to sequence
```

---

## Demo Projects

### 1. Sorting (demo.ipynb)
Teaches GPT to sort lists of numbers: `[2, 0, 1] → [0, 1, 2]`
- Shows that GPT can learn algorithmic tasks
- Uses gpt-nano (~90K params), trains in ~1 minute

### 2. CharGPT (projects/chargpt/)
Character-level language model on any text file (e.g., Shakespeare)
- Uses gpt-mini (~1M params)
- Generates new text that mimics the training data style

---

## Run the Examples

```bash
python trainer_explained.py    # Annotated training loop
python generate_text.py        # Text generation from scratch
```

**Requires:** `pip install torch`
