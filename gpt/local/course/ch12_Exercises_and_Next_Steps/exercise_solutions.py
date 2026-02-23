"""
Chapter 12: Exercise Solution Starters

This file contains STARTER CODE for some of the exercises.
Complete the TODO sections to solve each exercise.
"""

import math
import random

random.seed(42)

# ============================================================
# Value class (from Chapter 03) — needed for exercises
# ============================================================
class Value:
    __slots__ = ('data', 'grad', '_children', '_local_grads')
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
    def __pow__(self, n): return Value(self.data**n, (self,), (n * self.data**(n-1),))
    def exp(self): return Value(math.exp(self.data), (self,), (math.exp(self.data),))
    def log(self): return Value(math.log(self.data), (self,), (1/self.data,))
    def relu(self): return Value(max(0, self.data), (self,), (float(self.data > 0),))
    def __neg__(self): return self * -1
    def __radd__(self, other): return self + other
    def __sub__(self, other): return self + (-other)
    def __rmul__(self, other): return self * other
    def __truediv__(self, other): return self * other**-1
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
# EXERCISE 3: Add tanh() and sigmoid() to Value class
# ============================================================
print("=" * 50)
print("Exercise 3: Extending the Value Class")
print("=" * 50)

def tanh(self):
    """
    TODO: Implement tanh for the Value class.
    
    tanh(x) = (exp(2x) - 1) / (exp(2x) + 1)
    d(tanh(x))/dx = 1 - tanh(x)^2
    
    Hint: You can compute the forward value using math.tanh()
    and the local gradient using the derivative formula above.
    """
    t = math.tanh(self.data)
    return Value(t, (self,), (1 - t**2,))

# Attach to Value class
Value.tanh = tanh

# Test it
x = Value(0.5)
y = x.tanh()
y.backward()
print(f"  tanh({x.data}) = {y.data:.6f}")
print(f"  d(tanh)/dx = {x.grad:.6f}")
print(f"  Expected: tanh(0.5) = {math.tanh(0.5):.6f}")
print(f"  Expected grad: 1 - tanh(0.5)^2 = {1 - math.tanh(0.5)**2:.6f}")


def sigmoid(self):
    """
    TODO: Implement sigmoid for the Value class.
    
    sigmoid(x) = 1 / (1 + exp(-x))
    d(sigmoid(x))/dx = sigmoid(x) * (1 - sigmoid(x))
    """
    s = 1.0 / (1.0 + math.exp(-self.data))
    return Value(s, (self,), (s * (1 - s),))

Value.sigmoid = sigmoid

x = Value(1.0)
y = x.sigmoid()
y.backward()
s = 1.0 / (1.0 + math.exp(-1.0))
print(f"\n  sigmoid({x.data}) = {y.data:.6f}")
print(f"  d(sigmoid)/dx = {x.grad:.6f}")
print(f"  Expected: sigmoid(1.0) = {s:.6f}")
print(f"  Expected grad: {s * (1 - s):.6f}")


# ============================================================
# EXERCISE 4: Temperature Explorer
# ============================================================
print("\n" + "=" * 50)
print("Exercise 4: Temperature Explorer")
print("=" * 50)

def softmax(logits):
    max_val = max(logits)
    exps = [math.exp(v - max_val) for v in logits]
    total = sum(exps)
    return [e / total for e in exps]

logits = [3.0, 1.5, 0.5, 0.1, -0.5]
tokens = ['A', 'B', 'C', 'D', 'E']

print(f"\nLogits: {logits}")
print(f"Tokens: {tokens}\n")

temperatures = [0.1, 0.25, 0.5, 1.0, 2.0, 5.0]

print(f"{'Temp':>6} | ", end="")
for tok in tokens:
    print(f"  {tok:>5}", end="")
print(f" | {'Entropy':>8} | Description")
print("-" * 70)

for temp in temperatures:
    scaled = [l / temp for l in logits]
    probs = softmax(scaled)

    # Compute entropy: -sum(p * log(p))
    entropy = -sum(p * math.log(p + 1e-10) for p in probs)

    desc = ""
    if temp < 0.3:
        desc = "Nearly deterministic"
    elif temp < 0.8:
        desc = "Confident"
    elif temp <= 1.2:
        desc = "Balanced"
    elif temp < 3.0:
        desc = "Creative / random"
    else:
        desc = "Nearly uniform"

    print(f"{temp:>6.2f} | ", end="")
    for p in probs:
        print(f"  {p:>5.3f}", end="")
    print(f" | {entropy:>8.4f} | {desc}")

print("""
  Low temperature → low entropy → model picks the "best" token
  High temperature → high entropy → model picks more randomly
  
  Temperature = 1.0 is the "natural" distribution
  Temperature → 0 approaches argmax (greedy decoding)
  Temperature → ∞ approaches uniform random
""")


# ============================================================
# EXERCISE 1 STARTER: Bigram Model with Gradient Descent
# ============================================================
print("=" * 50)
print("Exercise 1 Starter: Bigram Model with Gradient Descent")
print("=" * 50)

# Dataset
names = ["emma", "olivia", "ava", "sophia", "mia", "luna", "ella", "aria"]
chars = sorted(set(''.join(names)))
BOS = len(chars)
vocab_size = len(chars) + 1

print(f"\nVocab: {chars + ['<BOS>']}")
print(f"Vocab size: {vocab_size}")

# The model: a weight matrix W where W[i][j] = score for token j following token i
W = [[Value(random.gauss(0, 0.5)) for _ in range(vocab_size)]
     for _ in range(vocab_size)]
params = [p for row in W for p in row]

def softmax_value(logits):
    max_val = max(v.data for v in logits)
    exps = [(v - max_val).exp() for v in logits]
    total = sum(exps)
    return [e / total for e in exps]

# TODO: Complete the training loop
# Hint: For each name, create pairs (current_char, next_char)
# and minimize cross-entropy loss

lr = 0.1
for step in range(200):
    total_loss = Value(0)
    count = 0

    for name in names:
        tokens = [BOS] + [chars.index(ch) for ch in name] + [BOS]
        for i in range(len(tokens) - 1):
            current = tokens[i]
            target = tokens[i + 1]
            logits = W[current]
            probs = softmax_value(logits)
            total_loss = total_loss + (-probs[target].log())
            count += 1

    avg_loss = total_loss * (1.0 / count)
    avg_loss.backward()

    for p in params:
        p.data -= lr * p.grad
        p.grad = 0

    if step % 20 == 0:
        print(f"  Step {step:3d} | Loss: {avg_loss.data:.4f}")

# Generate!
print("\n  Generated names:")
for i in range(10):
    name = []
    current = BOS
    for _ in range(20):
        logits = W[current]
        probs = softmax_value(logits)
        weights = [p.data for p in probs]
        current = random.choices(range(vocab_size), weights=weights, k=1)[0]
        if current == BOS:
            break
        name.append(chars[current])
    print(f"    {i+1:2d}. {''.join(name)}")

print("""
=== Next Steps ===

1. Try all the exercises above
2. Read the original source code with your new understanding
3. Watch Karpathy's YouTube videos for visual walkthroughs
4. Build something with minGPT — train on YOUR data!
5. Graduate to nanoGPT for real-world experiments

You now understand how GPT works, from the ground up.
The rest is just scale. Good luck!
""")
