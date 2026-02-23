# Chapter 03: Autograd — Teaching Computers Calculus

## Why Do We Need Calculus?

To train a model, we need to answer: **"If I nudge this parameter a tiny bit, does the output get better or worse?"** That's what a derivative tells us.

But a GPT has millions of parameters and hundreds of operations chained together. Computing derivatives by hand is impossible. So we use **automatic differentiation** (autograd).

---

## The Chain Rule — The One Rule That Powers All of AI

If you compute `y = f(g(x))`, the derivative of y with respect to x is:

```
dy/dx = dy/dg * dg/dx
```

"Multiply the local derivatives along the chain." That's it. Autograd just automates this.

---

## microGPT's `Value` Class

microGPT implements autograd from scratch in ~40 lines. The `Value` class:
1. Wraps a number
2. Records what operations created it (the computation graph)
3. Can compute gradients automatically via `backward()`

---

## The Computation Graph

When you write `c = a + b`, autograd secretly builds a graph:

```
a ──┐
    ├──(+)──→ c
b ──┘
```

Each node knows its children and the local derivative of the operation.

---

## Forward Pass vs Backward Pass

- **Forward pass**: Compute the output value (follow the arrows forward)
- **Backward pass**: Compute gradients by walking the graph backward, multiplying local derivatives (chain rule)

---

## Run the Examples

```bash
python value_basics.py        # Understand the Value class
python computation_graph.py   # See the chain rule in action
python gradient_descent.py    # Use gradients to optimize a simple function
```
