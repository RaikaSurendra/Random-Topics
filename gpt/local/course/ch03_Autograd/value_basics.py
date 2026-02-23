"""
Chapter 03: Understanding the Value Class

The Value class is microGPT's autograd engine.
It wraps a number and tracks the computation graph
so gradients can be computed automatically.

This is a simplified version of the Value class from microgpt.py
"""

import math

# ============================================================
# THE VALUE CLASS (simplified from microGPT)
# ============================================================
class Value:
    """
    A single scalar value that tracks its computation history.
    
    Think of it as a "smart number" that remembers:
    - Its current value (self.data)
    - How it was created (self._children, self._local_grads)
    - Its gradient, computed later (self.grad)
    """

    def __init__(self, data, children=(), local_grads=()):
        self.data = data                # the actual number
        self.grad = 0                   # derivative of loss w.r.t. this value
        self._children = children       # what values were used to compute this
        self._local_grads = local_grads # derivative of this op w.r.t. each child

    def __add__(self, other):
        """Addition: c = a + b"""
        other = other if isinstance(other, Value) else Value(other)
        # d(a+b)/da = 1, d(a+b)/db = 1
        return Value(self.data + other.data, (self, other), (1, 1))

    def __mul__(self, other):
        """Multiplication: c = a * b"""
        other = other if isinstance(other, Value) else Value(other)
        # d(a*b)/da = b, d(a*b)/db = a
        return Value(self.data * other.data, (self, other), (other.data, self.data))

    def __pow__(self, power):
        """Power: c = a ** n"""
        # d(a^n)/da = n * a^(n-1)
        return Value(self.data ** power, (self,), (power * self.data ** (power - 1),))

    def __neg__(self): return self * -1
    def __sub__(self, other): return self + (-other)
    def __truediv__(self, other): return self * other ** -1
    def __radd__(self, other): return self + other
    def __rmul__(self, other): return self * other

    def backward(self):
        """Compute gradients for all values in the computation graph."""
        # Step 1: Build topological ordering (children before parents)
        topo = []
        visited = set()
        def build_topo(v):
            if v not in visited:
                visited.add(v)
                for child in v._children:
                    build_topo(child)
                topo.append(v)
        build_topo(self)

        # Step 2: Backpropagate gradients
        self.grad = 1  # d(self)/d(self) = 1
        for v in reversed(topo):
            for child, local_grad in zip(v._children, v._local_grads):
                child.grad += local_grad * v.grad  # Chain rule!

    def __repr__(self):
        return f"Value(data={self.data:.4f}, grad={self.grad:.4f})"


# ============================================================
# EXAMPLE 1: Simple addition
# ============================================================
print("=== Example 1: Addition (c = a + b) ===")
a = Value(3.0)
b = Value(5.0)
c = a + b  # c = 8.0

c.backward()

print(f"  a = {a}")
print(f"  b = {b}")
print(f"  c = a + b = {c}")
print(f"  dc/da = {a.grad} (if we increase a by 1, c increases by 1)")
print(f"  dc/db = {b.grad} (if we increase b by 1, c increases by 1)")

# ============================================================
# EXAMPLE 2: Multiplication
# ============================================================
print("\n=== Example 2: Multiplication (c = a * b) ===")
a = Value(3.0)
b = Value(5.0)
c = a * b  # c = 15.0

c.backward()

print(f"  a = {a}")
print(f"  b = {b}")
print(f"  c = a * b = {c}")
print(f"  dc/da = {a.grad} (= b's value, because d(a*b)/da = b)")
print(f"  dc/db = {b.grad} (= a's value, because d(a*b)/db = a)")

# ============================================================
# EXAMPLE 3: A chain of operations (chain rule!)
# ============================================================
print("\n=== Example 3: Chain Rule (d = (a + b) * c) ===")
a = Value(2.0)
b = Value(3.0)
c = Value(4.0)

# Forward pass:
ab = a + b    # ab = 5.0
d = ab * c    # d = 20.0

# Backward pass:
d.backward()

print(f"  a = {a}")
print(f"  b = {b}")
print(f"  c = {c}")
print(f"  d = (a + b) * c = {d}")
print(f"  dd/da = {a.grad} (= c, because d = (a+b)*c, dd/da = c = 4)")
print(f"  dd/db = {b.grad} (= c, same reasoning)")
print(f"  dd/dc = {c.grad} (= a+b = 5)")

# ============================================================
# EXAMPLE 4: More complex expression
# ============================================================
print("\n=== Example 4: Complex Expression (y = (2x + 3)^2) ===")
x = Value(1.0)

# Forward pass:
y = (2 * x + 3) ** 2  # y = (2*1+3)^2 = 25

# Backward pass:
y.backward()

print(f"  x = {x}")
print(f"  y = (2x + 3)^2 = {y}")
print(f"  dy/dx = {x.grad}")
print(f"  By hand: dy/dx = 2 * 2 * (2x+3) = 4 * (2*1+3) = 20 ✓")

# ============================================================
# KEY TAKEAWAY
# ============================================================
print("""
=== Key Takeaway ===
The Value class lets us:
  1. Write normal math expressions (forward pass)
  2. Automatically compute ALL derivatives (backward pass)

In GPT, the expression is HUGE (millions of operations),
but the same principle applies. The chain rule just propagates
gradients through the entire computation graph.

Next: See computation_graph.py for a visual explanation.
""")
