"""
Chapter 08: Side-by-Side — Manual Python vs PyTorch

Shows the SAME operation implemented both ways,
so you can see exactly what PyTorch is doing for you.

Requires: pip install torch
"""

import math
import random

random.seed(42)

try:
    import torch
    import torch.nn as nn
    from torch.nn import functional as F
    torch.manual_seed(42)
except ImportError:
    print("PyTorch not installed. Run: pip install torch")
    exit(1)

# ============================================================
# COMPARISON 1: Softmax
# ============================================================
print("=" * 60)
print("Comparison 1: Softmax")
print("=" * 60)

scores = [2.0, 1.0, 0.5]

# MANUAL (like microGPT)
max_val = max(scores)
exps = [math.exp(s - max_val) for s in scores]
total = sum(exps)
manual_probs = [e / total for e in exps]

# PYTORCH (like minGPT)
torch_probs = F.softmax(torch.tensor(scores), dim=-1)

print(f"  Input scores: {scores}")
print(f"  Manual: [{', '.join(f'{p:.4f}' for p in manual_probs)}]")
print(f"  PyTorch: {torch_probs.tolist()}")
print(f"  Match: {all(abs(a - b) < 1e-6 for a, b in zip(manual_probs, torch_probs.tolist()))}")

# ============================================================
# COMPARISON 2: Linear Layer
# ============================================================
print("\n" + "=" * 60)
print("Comparison 2: Linear Layer (Matrix Multiply)")
print("=" * 60)

x = [1.0, 2.0, 3.0]
W = [[0.1, 0.2, 0.3],
     [0.4, 0.5, 0.6]]

# MANUAL (like microGPT)
manual_out = [sum(wi * xi for wi, xi in zip(row, x)) for row in W]

# PYTORCH (like minGPT)
x_t = torch.tensor(x)
W_t = torch.tensor(W)
torch_out = (W_t @ x_t).tolist()

print(f"  Input: {x}")
print(f"  Manual: {manual_out}")
print(f"  PyTorch: {torch_out}")
print(f"  Match: {all(abs(a - b) < 1e-6 for a, b in zip(manual_out, torch_out))}")

# ============================================================
# COMPARISON 3: Autograd (Gradient Computation)
# ============================================================
print("\n" + "=" * 60)
print("Comparison 3: Autograd")
print("=" * 60)

# MANUAL Value class (like microGPT)
class Value:
    def __init__(self, data, children=(), local_grads=()):
        self.data = data
        self.grad = 0
        self._children = children
        self._local_grads = local_grads
    def __add__(self, other):
        other = other if isinstance(other, Value) else Value(other)
        return Value(self.data + other.data, (self, other), (1, 1))
    def __mul__(self, other):
        other = other if isinstance(other, Value) else Value(other)
        return Value(self.data * other.data, (self, other), (other.data, self.data))
    def __pow__(self, n):
        return Value(self.data ** n, (self,), (n * self.data ** (n-1),))
    def __rmul__(self, other): return self * other
    def backward(self):
        topo, visited = [], set()
        def build(v):
            if v not in visited:
                visited.add(v)
                for c in v._children: build(c)
                topo.append(v)
        build(self)
        self.grad = 1
        for v in reversed(topo):
            for child, lg in zip(v._children, v._local_grads):
                child.grad += lg * v.grad

# Manual autograd
a_manual = Value(2.0)
b_manual = Value(3.0)
c_manual = (a_manual * b_manual + 5) ** 2  # (2*3+5)^2 = 121
c_manual.backward()

# PyTorch autograd
a_torch = torch.tensor(2.0, requires_grad=True)
b_torch = torch.tensor(3.0, requires_grad=True)
c_torch = (a_torch * b_torch + 5) ** 2
c_torch.backward()

print(f"  Expression: (a*b + 5)^2 where a=2, b=3")
print(f"  Result:     manual={c_manual.data:.1f}, pytorch={c_torch.item():.1f}")
print(f"  da:         manual={a_manual.grad:.1f}, pytorch={a_torch.grad.item():.1f}")
print(f"  db:         manual={b_manual.grad:.1f}, pytorch={b_torch.grad.item():.1f}")

# ============================================================
# COMPARISON 4: Cross-Entropy Loss
# ============================================================
print("\n" + "=" * 60)
print("Comparison 4: Cross-Entropy Loss")
print("=" * 60)

logits = [2.0, 1.0, 0.5]
target = 0  # correct token is index 0

# MANUAL (like microGPT)
max_val = max(logits)
exps = [math.exp(v - max_val) for v in logits]
total = sum(exps)
probs = [e / total for e in exps]
manual_loss = -math.log(probs[target])

# PYTORCH (like minGPT)
logits_t = torch.tensor([logits])  # add batch dim
target_t = torch.tensor([target])
torch_loss = F.cross_entropy(logits_t, target_t)

print(f"  Logits: {logits}, Target: {target}")
print(f"  Manual loss: {manual_loss:.6f}")
print(f"  PyTorch loss: {torch_loss.item():.6f}")
print(f"  Match: {abs(manual_loss - torch_loss.item()) < 1e-5}")

# ============================================================
# COMPARISON 5: Speed
# ============================================================
print("\n" + "=" * 60)
print("Comparison 5: Speed (rough estimate)")
print("=" * 60)

import time

size = 100

# Manual matrix multiply
W_manual = [[random.random() for _ in range(size)] for _ in range(size)]
x_manual = [random.random() for _ in range(size)]

start = time.time()
for _ in range(100):
    result = [sum(wi * xi for wi, xi in zip(row, x_manual)) for row in W_manual]
manual_time = time.time() - start

# PyTorch matrix multiply
W_torch = torch.randn(size, size)
x_torch = torch.randn(size)

start = time.time()
for _ in range(100):
    result = W_torch @ x_torch
pytorch_time = time.time() - start

speedup = manual_time / max(pytorch_time, 1e-10)
print(f"  {size}x{size} matrix-vector multiply × 100 iterations:")
print(f"  Manual Python: {manual_time*1000:.1f}ms")
print(f"  PyTorch:       {pytorch_time*1000:.1f}ms")
print(f"  Speedup:       ~{speedup:.0f}x")
print(f"  (On GPU, the speedup would be 100-1000x more!)")

print("""
=== Summary ===

microGPT and minGPT implement the SAME algorithm.
The difference is purely in efficiency:

  microGPT: every operation is a Python function call
  minGPT: operations are batched into fast tensor ops

This means:
  - microGPT trains for minutes on tiny data
  - minGPT can train on Shakespeare, add numbers, sort lists
  - Real GPT-2 trains on the entire internet

The math is identical. The speed is not.
""")
