# Chapter 04: Neural Network Building Blocks

## Overview

A neural network is built from a few simple, reusable pieces. Understanding these pieces is the key to understanding GPT.

---

## The Building Blocks

### 1. Linear Layer (Matrix Multiply)
The workhorse of neural networks. Takes an input vector, multiplies by a weight matrix, producing an output vector.

```
input:  [x1, x2, x3]      (3 numbers)
weights: 2x3 matrix        (6 numbers, learned)
output: [y1, y2]           (2 numbers)

y1 = w11*x1 + w12*x2 + w13*x3
y2 = w21*x1 + w22*x2 + w23*x3
```

This is just a weighted sum — each output is a mix of all inputs.

### 2. Activation Functions (ReLU, GELU)
Without activations, stacking linear layers would just be one big linear layer. Activations add non-linearity so the network can learn complex patterns.

```
ReLU(x) = max(0, x)     # Simple: zero out negatives (used in microGPT)
GELU(x) ≈ x * sigmoid(1.7*x)  # Smoother version (used in minGPT)
```

### 3. Softmax
Converts a list of arbitrary numbers into probabilities (positive, sum to 1):

```
scores:       [2.0, 1.0, 0.5]
softmax:      [0.59, 0.22, 0.19]  (sum = 1.0)
```

Used at the end of GPT to get "probability of each next token."

### 4. Layer Normalization / RMSNorm
Keeps numbers in a stable range so the network doesn't explode or vanish during training.

```
RMSNorm: scale each vector so its average squared value is ~1
LayerNorm: shift and scale so mean=0, variance=1
```

microGPT uses RMSNorm, minGPT uses LayerNorm.

### 5. Embedding
A lookup table: token ID → vector of numbers.

```
Token 5 → [0.12, -0.34, 0.56, ...]  (a vector of n_embd numbers)
```

This is how tokens enter the neural network — each token gets mapped to a dense vector.

---

## Run the Examples

```bash
python building_blocks.py   # Interactive demo of each building block
python mlp.py               # A complete Multi-Layer Perceptron (MLP)
```
