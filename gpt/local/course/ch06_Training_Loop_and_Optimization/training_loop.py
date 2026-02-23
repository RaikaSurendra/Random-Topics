"""
Chapter 06: A Complete Mini Training Loop

This script trains a tiny neural network on a simple pattern,
using the EXACT same training loop structure as microGPT.

The pattern: learn to predict the next character in "abcabc..."
"""

import math
import random

random.seed(42)

# ============================================================
# Autograd engine (from Chapter 03)
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
    def __neg__(self): return self * -1
    def __sub__(self, other): return self + (-other)
    def __radd__(self, other): return self + other
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
# Simple model: a single linear layer (logits = W @ one_hot_input)
# This is basically just a lookup table — the simplest "model"
# ============================================================
vocab = ['a', 'b', 'c']
vocab_size = len(vocab)
char_to_id = {ch: i for i, ch in enumerate(vocab)}

# The training data: repeating "abc"
data = "abcabcabcabcabc"

# Our "model": a weight matrix that maps each token to logits for next token
# W[i][j] = score for predicting token j when input is token i
matrix = lambda rows, cols: [[Value(random.gauss(0, 0.5)) for _ in range(cols)]
                              for _ in range(rows)]
W = matrix(vocab_size, vocab_size)
params = [p for row in W for p in row]

def softmax(logits):
    max_val = max(val.data for val in logits)
    exps = [(val - max_val).exp() for val in logits]
    total = sum(exps)
    return [e / total for e in exps]

# ============================================================
# Training loop — same structure as microGPT!
# ============================================================
print("=" * 50)
print("Training a tiny model to predict: a→b, b→c, c→a")
print(f"Parameters: {len(params)}")
print("=" * 50)

learning_rate = 0.5
beta1, beta2, eps = 0.85, 0.99, 1e-8
m = [0.0] * len(params)
v = [0.0] * len(params)

num_steps = 100

for step in range(num_steps):
    # 1. Pick a training example (input token → target token)
    pos = step % (len(data) - 1)
    input_id = char_to_id[data[pos]]
    target_id = char_to_id[data[pos + 1]]

    # 2. Forward pass: look up the row, get logits, compute loss
    logits = W[input_id]  # simple lookup: logits for this input
    probs = softmax(logits)
    loss = -probs[target_id].log()

    # 3. Backward pass
    loss.backward()

    # 4. Adam optimizer update
    lr_t = learning_rate * (1 - step / num_steps)  # linear decay
    for i, p in enumerate(params):
        m[i] = beta1 * m[i] + (1 - beta1) * p.grad
        v[i] = beta2 * v[i] + (1 - beta2) * p.grad ** 2
        m_hat = m[i] / (1 - beta1 ** (step + 1))
        v_hat = v[i] / (1 - beta2 ** (step + 1))
        p.data -= lr_t * m_hat / (v_hat ** 0.5 + eps)
        p.grad = 0  # reset gradient

    if step % 10 == 0:
        print(f"  Step {step:3d} | loss {loss.data:.4f} | "
              f"input='{data[pos]}' target='{data[pos+1]}' "
              f"P(correct)={probs[target_id].data:.3f}")

# ============================================================
# Inference: what did the model learn?
# ============================================================
print("\n" + "=" * 50)
print("After Training: What the Model Learned")
print("=" * 50)

print("\nPrediction probabilities:")
for input_ch in vocab:
    input_id = char_to_id[input_ch]
    logits = W[input_id]
    probs = softmax(logits)
    print(f"\n  Given '{input_ch}', predict:")
    for j, ch in enumerate(vocab):
        bar = "█" * int(probs[j].data * 30)
        print(f"    '{ch}': {probs[j].data:.3f} {bar}")

# Generate a sequence!
print("\n" + "=" * 50)
print("Generating text (starting from 'a'):")
print("=" * 50)

current = char_to_id['a']
generated = ['a']
for _ in range(20):
    logits = W[current]
    probs = softmax(logits)
    # Greedy: pick the highest probability token
    next_token = max(range(vocab_size), key=lambda i: probs[i].data)
    generated.append(vocab[next_token])
    current = next_token

print(f"  {''.join(generated)}")
print(f"  (Should be 'abcabcabc...' if training worked!)")

print("""
=== This Is microGPT's Training Loop ===

What we just did:
  1. Pick a document, tokenize it                    ← line 156-158
  2. For each position, predict next token            ← line 163-168
  3. Compute cross-entropy loss                       ← line 167-169
  4. Backward pass to get gradients                   ← line 172
  5. Adam optimizer updates all parameters            ← line 176-182

The only difference in real microGPT:
  - The model is a full transformer (not a lookup table)
  - It processes entire sequences, not single tokens
  - It has ~5000 parameters instead of 9

But the training loop structure is IDENTICAL.
""")
