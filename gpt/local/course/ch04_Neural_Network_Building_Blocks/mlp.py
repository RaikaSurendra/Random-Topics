"""
Chapter 04: The MLP (Multi-Layer Perceptron)

The MLP is a key component inside every Transformer block.
It processes each token position independently, adding
"thinking capacity" to the model.

Structure: Linear → ReLU → Linear
"""

import math
import random

random.seed(42)

# ============================================================
# Value class (from Chapter 03) for automatic gradients
# ============================================================
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
    def relu(self):
        return Value(max(0, self.data), (self,), (float(self.data > 0),))
    def __neg__(self): return self * -1
    def __sub__(self, other): return self + (-other)
    def __radd__(self, other): return self + other
    def __rmul__(self, other): return self * other
    def __truediv__(self, other): return self * other ** -1

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
# MLP: Two linear layers with ReLU in between
# ============================================================
# In GPT, the MLP has this structure:
#   input (n_embd) → Linear (4*n_embd) → ReLU → Linear (n_embd) → output
# It "expands" the representation, processes it, then "compresses" back.

n_in = 3        # input dimension
n_hidden = 8    # hidden dimension (expanded)
n_out = 3       # output dimension (same as input in GPT)

# Initialize weights randomly
def make_matrix(rows, cols):
    return [[Value(random.gauss(0, 0.3)) for _ in range(cols)] for _ in range(rows)]

W1 = make_matrix(n_hidden, n_in)   # first linear layer
W2 = make_matrix(n_out, n_hidden)  # second linear layer

def linear(x, w):
    return [sum(wi * xi for wi, xi in zip(row, x)) for row in w]

def mlp_forward(x):
    """The MLP forward pass: Linear → ReLU → Linear"""
    # Step 1: Expand (3 → 8)
    h = linear(x, W1)
    # Step 2: Activate (apply ReLU)
    h = [hi.relu() for hi in h]
    # Step 3: Compress (8 → 3)
    out = linear(h, W2)
    return out

# ============================================================
# Demo: Run a vector through the MLP
# ============================================================
print("=" * 50)
print("MLP (Multi-Layer Perceptron) Demo")
print("=" * 50)

x = [Value(1.0), Value(0.5), Value(-0.3)]

print(f"\nInput ({n_in} values):  [{', '.join(f'{v.data:.2f}' for v in x)}]")

h = linear(x, W1)
print(f"After Linear1 ({n_hidden} values): [{', '.join(f'{v.data:.2f}' for v in h)}]")

h_relu = [hi.relu() for hi in h]
print(f"After ReLU ({n_hidden} values):    [{', '.join(f'{v.data:.2f}' for v in h_relu)}]")

out = linear(h_relu, W2)
print(f"After Linear2 ({n_out} values): [{', '.join(f'{v.data:.2f}' for v in out)}]")

# ============================================================
# Train the MLP to learn a simple function
# ============================================================
print("\n" + "=" * 50)
print("Training an MLP to learn: [a, b, c] → [a+b, b+c, a+c]")
print("=" * 50)

# Training data
data = [
    ([1.0, 2.0, 3.0], [3.0, 5.0, 4.0]),
    ([0.5, 1.0, 0.5], [1.5, 1.5, 1.0]),
    ([2.0, 0.0, 1.0], [2.0, 1.0, 3.0]),
    ([1.0, 1.0, 1.0], [2.0, 2.0, 2.0]),
    ([0.0, 3.0, 2.0], [3.0, 5.0, 2.0]),
]

# Collect all parameters for gradient descent
params = [p for row in W1 for p in row] + [p for row in W2 for p in row]
print(f"Number of parameters: {len(params)}")

lr = 0.005
for step in range(200):
    # Forward pass on all data points
    total_loss = Value(0)
    for x_data, y_target in data:
        x = [Value(v) for v in x_data]
        y_pred = mlp_forward(x)
        for yp, yt in zip(y_pred, y_target):
            total_loss = total_loss + (yp - yt) ** 2

    total_loss = total_loss * (1.0 / (len(data) * n_out))

    # Backward pass
    total_loss.backward()

    # Update parameters
    for p in params:
        p.data -= lr * p.grad
        p.grad = 0

    if step % 20 == 0:
        print(f"  Step {step:3d} | Loss: {total_loss.data:.4f}")

# Test
print("\nAfter training:")
for x_data, y_target in data[:3]:
    x = [Value(v) for v in x_data]
    y_pred = mlp_forward(x)
    pred = [f"{v.data:.2f}" for v in y_pred]
    target = [f"{v:.2f}" for v in y_target]
    print(f"  Input: {x_data} → Predicted: [{', '.join(pred)}] | Target: [{', '.join(target)}]")

print("""
=== In GPT ===
Every transformer block contains an MLP just like this.
  - Input dimension: n_embd (e.g., 768 for GPT-2)
  - Hidden dimension: 4 * n_embd (e.g., 3072)
  - The MLP processes each token position independently
  - It adds "thinking" — the ability to transform representations

The MLP is the "feedforward" part. Next: Attention is the "communication" part.
""")
