"""
Chapter 03: Visualizing the Computation Graph

This script shows how operations build a graph,
and how backward() walks it in reverse to compute gradients.
"""

import math

class Value:
    def __init__(self, data, children=(), local_grads=(), label=''):
        self.data = data
        self.grad = 0
        self._children = children
        self._local_grads = local_grads
        self.label = label

    def __add__(self, other):
        other = other if isinstance(other, Value) else Value(other)
        return Value(self.data + other.data, (self, other), (1, 1),
                     f"({self.label}+{other.label})")

    def __mul__(self, other):
        other = other if isinstance(other, Value) else Value(other)
        return Value(self.data * other.data, (self, other), (other.data, self.data),
                     f"({self.label}*{other.label})")

    def __pow__(self, n):
        return Value(self.data ** n, (self,), (n * self.data ** (n-1),),
                     f"({self.label}^{n})")

    def __neg__(self): return self * Value(-1, label='-1')
    def __sub__(self, other): return self + (-other)
    def __rmul__(self, other): return Value(other, label=str(other)) * self

    def log(self):
        return Value(math.log(self.data), (self,), (1/self.data,),
                     f"log({self.label})")

    def exp(self):
        return Value(math.exp(self.data), (self,), (math.exp(self.data),),
                     f"exp({self.label})")

    def backward(self):
        topo = []
        visited = set()
        def build_topo(v):
            if v not in visited:
                visited.add(v)
                for child in v._children:
                    build_topo(child)
                topo.append(v)
        build_topo(self)
        self.grad = 1
        for v in reversed(topo):
            for child, local_grad in zip(v._children, v._local_grads):
                child.grad += local_grad * v.grad


# ============================================================
# BUILD A COMPUTATION: loss = -log(softmax(score)[target])
# This is EXACTLY what happens in GPT at each prediction step!
# ============================================================

print("=" * 60)
print("Computation Graph: A Mini Cross-Entropy Loss")
print("=" * 60)
print()
print("Scenario: Model outputs scores [2.0, 1.0, 0.5] for 3 tokens.")
print("The correct token is index 0. What's the loss?\n")

# These are our "model outputs" (logits)
s0 = Value(2.0, label='s0')   # score for token 0 (correct answer)
s1 = Value(1.0, label='s1')   # score for token 1
s2 = Value(0.5, label='s2')   # score for token 2

# Softmax: convert scores to probabilities
# softmax(si) = exp(si) / sum(exp(sj))
e0 = s0.exp()
e1 = s1.exp()
e2 = s2.exp()
total = e0 + e1 + e2

p0 = e0 * (total ** -1)  # probability of token 0

# Cross-entropy loss for the correct token (index 0)
loss = -(p0.log())

print("Forward Pass (computing the loss):")
print(f"  Scores:        s0={s0.data:.2f}, s1={s1.data:.2f}, s2={s2.data:.2f}")
print(f"  Exponentials:  exp(s0)={e0.data:.4f}, exp(s1)={e1.data:.4f}, exp(s2)={e2.data:.4f}")
print(f"  Sum of exps:   {total.data:.4f}")
print(f"  Prob of s0:    {p0.data:.4f}  ({p0.data*100:.1f}%)")
print(f"  Loss:          -log({p0.data:.4f}) = {loss.data:.4f}")

# Now compute gradients!
loss.backward()

print(f"\nBackward Pass (computing gradients):")
print(f"  d(loss)/d(s0) = {s0.grad:.4f}")
print(f"  d(loss)/d(s1) = {s1.grad:.4f}")
print(f"  d(loss)/d(s2) = {s2.grad:.4f}")

print(f"""
Interpretation:
  s0.grad = {s0.grad:.4f} (NEGATIVE → increasing s0 DECREASES loss → good!)
  s1.grad = {s1.grad:.4f} (POSITIVE → increasing s1 INCREASES loss → bad!)
  s2.grad = {s2.grad:.4f} (POSITIVE → increasing s2 INCREASES loss → bad!)

This makes sense! The correct answer is token 0, so:
  - We WANT s0 to be higher (gradient pushes it up)
  - We WANT s1, s2 to be lower (gradient pushes them down)

This is exactly how a neural network learns:
  1. Forward pass → compute loss (how wrong are we?)
  2. Backward pass → compute gradients (which direction to adjust?)
  3. Update parameters in the direction that reduces loss
""")

# ============================================================
# VISUALIZE THE GRAPH STRUCTURE
# ============================================================
print("=" * 60)
print("The Computation Graph (text visualization)")
print("=" * 60)
print("""
     s0 ─→ exp() ─→ e0 ──┐
                          ├─→ e0/total ─→ p0 ─→ log() ─→ neg ─→ LOSS
     s1 ─→ exp() ─→ e1 ──┤                ↑
                          ├─→ total ───────┘
     s2 ─→ exp() ─→ e2 ──┘

Forward:  Left to right (compute values)
Backward: Right to left (compute gradients using chain rule)

Each arrow multiplies by the local derivative.
The chain rule = multiply all local derivatives along the path.
""")
