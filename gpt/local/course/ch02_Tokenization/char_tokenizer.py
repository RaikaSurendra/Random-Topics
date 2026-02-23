"""
Chapter 02: Character-Level Tokenizer

This is exactly how microGPT tokenizes text.
Each unique character becomes a token ID (integer).
"""

# ============================================================
# STEP 1: Build vocabulary from data
# ============================================================
# Imagine our dataset is a list of names
docs = ["emma", "olivia", "ava", "sophia", "isabella", "mia"]

# Find all unique characters, sorted
all_text = ''.join(docs)
uchars = sorted(set(all_text))

print("=== Vocabulary ===")
print(f"Unique characters: {uchars}")
print(f"Number of unique characters: {len(uchars)}")

# Each character maps to an integer index
char_to_id = {ch: i for i, ch in enumerate(uchars)}
id_to_char = {i: ch for i, ch in enumerate(uchars)}

print("\nCharacter → ID mapping:")
for ch, idx in char_to_id.items():
    print(f"  '{ch}' → {idx}")

# ============================================================
# STEP 2: Add special tokens
# ============================================================
# BOS = Beginning of Sequence. A special token that marks boundaries.
BOS = len(uchars)  # Gets the next available ID
vocab_size = len(uchars) + 1

print(f"\nBOS token ID: {BOS}")
print(f"Total vocab size: {vocab_size}")

# ============================================================
# STEP 3: Encode — turn text into numbers
# ============================================================
def encode(text):
    """Convert a string to a list of token IDs"""
    return [char_to_id[ch] for ch in text]

def decode(token_ids):
    """Convert a list of token IDs back to a string"""
    return ''.join(id_to_char[i] for i in token_ids if i != BOS)

# Try encoding some names
print("\n=== Encoding Examples ===")
for name in ["emma", "ava", "mia"]:
    tokens = encode(name)
    print(f"  '{name}' → {tokens}")

    # Add BOS tokens on both sides (like microGPT does)
    tokens_with_bos = [BOS] + tokens + [BOS]
    print(f"  with BOS: {tokens_with_bos}")

# ============================================================
# STEP 4: Decode — turn numbers back to text
# ============================================================
print("\n=== Decoding Examples ===")
sample_tokens = [3, 7, 7, 0]  # e, m, m, a
decoded = decode(sample_tokens)
print(f"  {sample_tokens} → '{decoded}'")

# ============================================================
# STEP 5: How this connects to the model
# ============================================================
print("""
=== How This Connects to GPT ===

During training, the model sees sequences like:
  [BOS, e, m, m, a, BOS]  →  [{bos}, {e}, {m}, {m2}, {a}, {bos2}]

At each position, it tries to predict the NEXT token:
  Given [BOS]           → predict 'e'  (token {e})
  Given [BOS, e]        → predict 'm'  (token {m})
  Given [BOS, e, m]     → predict 'm'  (token {m2})
  Given [BOS, e, m, m]  → predict 'a'  (token {a})
  Given [BOS, e, m, m, a] → predict BOS (end of name)

The model outputs {vocab_size} probabilities — one for each possible
next token. Training adjusts the model so the correct next token
gets the highest probability.
""".format(
    bos=BOS, e=char_to_id['e'], m=char_to_id['m'],
    m2=char_to_id['m'], a=char_to_id['a'],
    bos2=BOS, vocab_size=vocab_size
))
