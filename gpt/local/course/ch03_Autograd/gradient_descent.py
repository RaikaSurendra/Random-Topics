"""
Chapter 03: Gradient Descent — Using Gradients to Learn

This is the fundamental learning algorithm:
  1. Start with random parameter values
  2. Compute how wrong the output is (loss)
  3. Compute gradients (which direction to adjust)
  4. Nudge parameters to reduce the loss
  5. Repeat

We'll train a tiny "model" (just 2 parameters) to fit a target.
"""

import math
import random

random.seed(42)

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
    def __neg__(self): return self * -1
    def __sub__(self, other): return self + (-other)
    def __radd__(self, other): return self + other
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


# ============================================================
# PROBLEM: Learn y = 3x + 7
# Our model: y_pred = w*x + b  (we must find w=3, b=7)
# ============================================================

# Initialize parameters randomly
w = Value(random.uniform(-1, 1))  # will learn to be ~3.0
b = Value(random.uniform(-1, 1))  # will learn to be ~7.0

# Training data: (x, y) pairs from the true function y = 3x + 7
data = [(1, 10), (2, 13), (3, 16), (4, 19), (5, 22)]

learning_rate = 0.01

print("=" * 50)
print("Goal: Learn y = 3x + 7")
print(f"Starting: w = {w.data:.4f}, b = {b.data:.4f}")
print("=" * 50)

for step in range(100):
    # ---- Forward pass: compute predictions and loss ----
    total_loss = Value(0)
    for x_val, y_true in data:
        y_pred = w * x_val + b           # our model's prediction
        diff = y_pred - y_true            # error
        loss = diff ** 2                  # squared error (always positive)
        total_loss = total_loss + loss

    # Average loss
    avg_loss = total_loss * (1.0 / len(data))

    # ---- Backward pass: compute gradients ----
    avg_loss.backward()

    # ---- Update parameters: move in the opposite direction of gradient ----
    w.data -= learning_rate * w.grad
    b.data -= learning_rate * b.grad

    # ---- Reset gradients for next step ----
    w.grad = 0
    b.grad = 0

    # Print progress
    if step % 10 == 0 or step == 99:
        print(f"  Step {step:3d} | loss = {avg_loss.data:.4f} | w = {w.data:.4f} | b = {b.data:.4f}")

    # Recreate Value objects to reset computation graph
    w = Value(w.data)
    b = Value(b.data)

print(f"\n  Final: w = {w.data:.4f} (target: 3.0), b = {b.data:.4f} (target: 7.0)")

print("""
=== What Just Happened ===

1. We started with random w and b
2. Each step:
   a. Forward pass: computed predictions and loss
   b. Backward pass: autograd computed d(loss)/dw and d(loss)/db
   c. Update: moved w and b in the direction that reduces loss
3. After 100 steps, w ≈ 3.0 and b ≈ 7.0

This is EXACTLY what happens in GPT training:
  - Instead of 2 parameters (w, b), GPT has millions
  - Instead of y = wx + b, GPT is a transformer neural network
  - Instead of squared error, GPT uses cross-entropy loss
  - But the loop is identical: forward → backward → update

Next chapter: What are the building blocks of that neural network?
""")
