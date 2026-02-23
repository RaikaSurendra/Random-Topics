# Chapter 08: Scaling Up with PyTorch — Introduction to minGPT

## Why PyTorch?

microGPT is beautiful for learning, but it's **painfully slow**. Processing one name takes seconds because every single multiplication is a Python function call.

PyTorch solves this by:
1. **Tensors**: Multi-dimensional arrays that run on GPU (1000x faster math)
2. **Autograd**: Automatic differentiation built-in (no need for our `Value` class)
3. **nn.Module**: Clean way to organize model layers
4. **Optimizers**: Adam, SGD, etc. already implemented and optimized

---

## The Same Ideas, Better Tools

| Concept | microGPT (Pure Python) | minGPT (PyTorch) |
|---|---|---|
| Numbers | `Value(3.14)` | `torch.tensor(3.14)` |
| Gradients | `value.backward()` | `tensor.backward()` |
| Linear layer | `linear(x, w)` (manual loop) | `nn.Linear(in, out)` |
| Matrix multiply | Nested for-loops | `@` operator (GPU-accelerated) |
| Softmax | Manual exp/sum | `F.softmax(x)` |
| Optimizer | Manual Adam code | `torch.optim.AdamW(...)` |

---

## minGPT Project Structure

```
minGPT/
├── mingpt/
│   ├── model.py      ← The GPT model (311 lines)
│   ├── trainer.py     ← Training loop (110 lines)
│   ├── bpe.py         ← BPE tokenizer (320 lines)
│   └── utils.py       ← Config helpers (104 lines)
├── projects/
│   ├── chargpt/       ← Character-level text generation
│   └── adder/         ← Teaching GPT to add numbers
├── demo.ipynb         ← Sorting demo
└── generate.ipynb     ← GPT-2 text generation
```

---

## Key PyTorch Concepts for minGPT

### Tensors
```python
# A tensor is like a multi-dimensional array
x = torch.tensor([1.0, 2.0, 3.0])           # 1D
W = torch.randn(4, 3)                        # 2D (matrix)
batch = torch.randn(32, 10, 768)             # 3D (batch of sequences)
```

### nn.Module
```python
# Every layer inherits from nn.Module
class MyLayer(nn.Module):
    def __init__(self):
        super().__init__()
        self.linear = nn.Linear(768, 768)  # weight matrix inside
    
    def forward(self, x):
        return self.linear(x)
```

### Autograd
```python
x = torch.tensor(3.0, requires_grad=True)
y = x ** 2 + 2 * x
y.backward()        # computes dy/dx
print(x.grad)       # tensor(8.)  (= 2*3 + 2)
```

---

## Run the Examples

```bash
python pytorch_basics.py       # PyTorch fundamentals
python pytorch_vs_manual.py    # Side-by-side: manual vs PyTorch
```

**Note:** These examples require PyTorch. Install with: `pip install torch`
