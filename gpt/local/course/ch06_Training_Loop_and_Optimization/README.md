# Chapter 06: Training — How Models Learn

## The Training Loop

Every neural network trains with the same loop:

```
for each training step:
    1. Pick a training example
    2. Forward pass: run the model, compute the loss
    3. Backward pass: compute gradients (autograd)
    4. Update: adjust parameters to reduce the loss
```

---

## Loss Function: Cross-Entropy

For language models, the loss measures: **"How surprised was the model by the correct next token?"**

```
Model predicts: P("the")=0.6, P("a")=0.3, P("cat")=0.1
Correct answer: "the"

Loss = -log(0.6) = 0.51  (low loss — model was fairly confident)

Model predicts: P("the")=0.1, P("a")=0.1, P("cat")=0.8
Correct answer: "the"

Loss = -log(0.1) = 2.30  (high loss — model was wrong!)
```

Lower loss = better predictions. Training minimizes the average loss.

---

## The Optimizer: Adam

Simple gradient descent updates parameters as: `p -= learning_rate * gradient`

**Adam** is smarter — it maintains:
- **Momentum (m)**: Running average of gradients (direction smoothing)
- **Velocity (v)**: Running average of squared gradients (per-parameter learning rate)

This makes training faster and more stable.

---

## Learning Rate

The learning rate controls step size:
- **Too high**: Parameters overshoot, loss explodes
- **Too low**: Training is painfully slow
- **Just right**: Loss decreases steadily

Common practice: **start with a reasonable LR and decay it** over training.
microGPT uses linear decay: `lr = lr_start * (1 - step/total_steps)`

---

## Run the Examples

```bash
python cross_entropy.py     # Understanding the loss function
python adam_optimizer.py     # How Adam works, with visualization
python training_loop.py     # A complete mini training loop
```
