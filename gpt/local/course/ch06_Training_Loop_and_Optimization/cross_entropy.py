"""
Chapter 06: Cross-Entropy Loss — The GPT Loss Function

The cross-entropy loss measures how "surprised" the model is
by the correct answer. It's the standard loss function for
classification and language modeling.

Formula: loss = -log(probability_of_correct_token)
"""

import math

# ============================================================
# The Intuition
# ============================================================
print("=" * 60)
print("Cross-Entropy Loss: How 'Surprised' Is the Model?")
print("=" * 60)

print("""
If the model assigns HIGH probability to the correct answer:
  loss = -log(0.9) = {:.3f}  ← Low loss (good!)

If the model assigns LOW probability to the correct answer:
  loss = -log(0.1) = {:.3f}  ← High loss (bad!)

If the model assigns VERY LOW probability:
  loss = -log(0.01) = {:.3f} ← Very high loss (terrible!)
""".format(-math.log(0.9), -math.log(0.1), -math.log(0.01)))

# ============================================================
# Step by step: from logits to loss
# ============================================================
print("=" * 60)
print("From Model Output to Loss (Step by Step)")
print("=" * 60)

# Pretend our vocabulary is: [a, b, c, d, e] (5 tokens)
vocab = ['a', 'b', 'c', 'd', 'e']
correct_token = 2  # the correct next token is 'c'

# Step 1: Model outputs raw scores (logits)
logits = [1.0, 2.0, 4.0, 1.5, 0.5]
print(f"\nStep 1 - Model outputs logits (raw scores):")
for i, (tok, score) in enumerate(zip(vocab, logits)):
    marker = " ← correct" if i == correct_token else ""
    print(f"  '{tok}': {score:.1f}{marker}")

# Step 2: Softmax converts to probabilities
def softmax(logits):
    max_val = max(logits)
    exps = [math.exp(v - max_val) for v in logits]
    total = sum(exps)
    return [e / total for e in exps]

probs = softmax(logits)
print(f"\nStep 2 - Softmax → probabilities:")
for i, (tok, p) in enumerate(zip(vocab, probs)):
    bar = "█" * int(p * 40)
    marker = " ← correct" if i == correct_token else ""
    print(f"  '{tok}': {p:.4f} {bar}{marker}")
print(f"  Sum: {sum(probs):.4f}")

# Step 3: Cross-entropy loss
loss = -math.log(probs[correct_token])
print(f"\nStep 3 - Loss = -log(P(correct)) = -log({probs[correct_token]:.4f}) = {loss:.4f}")

# ============================================================
# What the gradients look like
# ============================================================
print("\n" + "=" * 60)
print("What Gradients Tell the Model")
print("=" * 60)
print(f"""
After computing loss, backpropagation gives us gradients.
For softmax + cross-entropy, the gradient for each logit is simple:

  gradient[i] = probability[i] - (1 if i is correct else 0)
""")

for i, (tok, p) in enumerate(zip(vocab, probs)):
    target = 1.0 if i == correct_token else 0.0
    grad = p - target
    direction = "↓ decrease" if grad > 0 else "↑ INCREASE"
    print(f"  '{tok}': grad = {p:.4f} - {target:.0f} = {grad:+.4f}  → {direction} this score")

print("""
The gradients push the model to:
  - INCREASE the score of the correct token ('c')
  - DECREASE the scores of all incorrect tokens
  - The adjustment is proportional to how wrong the model was
""")

# ============================================================
# Average loss over a sequence
# ============================================================
print("=" * 60)
print("Average Loss Over a Sequence")
print("=" * 60)

# Simulating predictions for "hello" where model gets better over positions
sequence = [
    ("h", [0.1, 0.8, 0.05, 0.05]),    # model correctly predicts 'h' with 10%
    ("e", [0.05, 0.05, 0.8, 0.1]),     # model correctly predicts 'e' with 80%
    ("l", [0.2, 0.6, 0.1, 0.1]),       # model correctly predicts 'l' with 60%
    ("l", [0.1, 0.1, 0.7, 0.1]),       # model correctly predicts second 'l' with 70%
    ("o", [0.1, 0.1, 0.1, 0.7]),       # model correctly predicts 'o' with 70%
]

print(f"\nPredicting each character in a sequence:")
losses = []
for char, probs_for_correct in sequence:
    p = probs_for_correct[0]  # simplified: first prob is for correct token
    l = -math.log(max(p, 1e-10))
    losses.append(l)
    print(f"  '{char}': P(correct) = {p:.2f}, loss = {l:.4f}")

avg_loss = sum(losses) / len(losses)
print(f"\n  Average loss = {avg_loss:.4f}")
print(f"  This is what microGPT computes: (1/n) * sum(losses)")
print(f"\n  Training goal: make this number as small as possible!")
