"""
Chapter 01: The Simplest Possible "Language Model"

This is NOT a real language model — it's a toy to show the core idea:
  1. Learn patterns from data (training)
  2. Generate new text using those patterns (inference)

We simply count how often each character follows another character.
"""

import random

# ============================================================
# STEP 1: Our "training data" — a few names
# ============================================================
training_data = [
    "emma", "olivia", "ava", "sophia", "isabella",
    "mia", "charlotte", "amelia", "harper", "evelyn",
]

print("=== Training Data ===")
for name in training_data:
    print(f"  {name}")

# ============================================================
# STEP 2: TRAINING — Count character transitions
# ============================================================
# We'll count: given character X, how often does character Y come next?
# We use a special character '.' to mean "start" or "end" of a name.

# Build a dictionary of dictionaries:
#   counts['a']['b'] = number of times 'b' follows 'a' in the training data
counts = {}

for name in training_data:
    # Add start/end markers: ".emma."
    chars = ['.'] + list(name) + ['.']
    for i in range(len(chars) - 1):
        current = chars[i]
        next_char = chars[i + 1]
        if current not in counts:
            counts[current] = {}
        counts[current][next_char] = counts[current].get(next_char, 0) + 1

# Let's look at what follows 'a':
print("\n=== What follows 'a' in training data? ===")
if 'a' in counts:
    total = sum(counts['a'].values())
    for char, count in sorted(counts['a'].items(), key=lambda x: -x[1]):
        prob = count / total
        display = "END" if char == '.' else char
        print(f"  '{display}' : {count} times ({prob:.0%})")

# ============================================================
# STEP 3: Convert counts to probabilities
# ============================================================
probs = {}
for current_char, next_chars in counts.items():
    total = sum(next_chars.values())
    probs[current_char] = {}
    for next_char, count in next_chars.items():
        probs[current_char][next_char] = count / total

# ============================================================
# STEP 4: INFERENCE — Generate new names!
# ============================================================
print("\n=== Generated Names (sampling from our 'model') ===")
random.seed(42)

for i in range(10):
    name = []
    current = '.'  # start token

    for _ in range(20):  # max length safety
        if current not in probs:
            break
        # Get possible next characters and their probabilities
        next_chars = list(probs[current].keys())
        weights = [probs[current][c] for c in next_chars]

        # Sample randomly, weighted by probability
        chosen = random.choices(next_chars, weights=weights, k=1)[0]

        if chosen == '.':  # end token
            break
        name.append(chosen)
        current = chosen

    print(f"  {i+1:2d}. {''.join(name)}")

# ============================================================
# KEY TAKEAWAYS
# ============================================================
print("""
=== Key Takeaways ===
1. We LEARNED patterns from data (counted character transitions)
2. We GENERATED new text by sampling from those patterns
3. The generated names "sound like" the training data but are new

This is exactly what GPT does — just with WAY more sophisticated
pattern detection (neural networks instead of simple counting).

Next chapter: How do we turn text into numbers? → Tokenization
""")
