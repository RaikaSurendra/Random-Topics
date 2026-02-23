"""
Chapter 02: BPE (Byte Pair Encoding) Intuition

BPE is the tokenizer used by GPT-2 and minGPT.
This simplified example shows the IDEA behind BPE.
(The real implementation is in minGPT/mingpt/bpe.py)
"""

# ============================================================
# THE IDEA: Repeatedly merge the most common pair of tokens
# ============================================================

def simple_bpe_train(corpus, num_merges):
    """
    Train a simple BPE tokenizer.
    
    Start with individual characters as tokens.
    Then repeatedly find the most common adjacent pair
    and merge them into a single new token.
    """
    # Start: each word is split into individual characters
    # We add a special end-of-word marker '_'
    words = {}
    for word in corpus.split():
        chars = ' '.join(list(word)) + ' _'
        words[chars] = words.get(chars, 0) + 1

    print("=== Starting tokens (individual characters) ===")
    print(f"  Words: {words}\n")

    merges = []

    for step in range(num_merges):
        # Count all adjacent pairs
        pairs = {}
        for word, freq in words.items():
            symbols = word.split()
            for i in range(len(symbols) - 1):
                pair = (symbols[i], symbols[i + 1])
                pairs[pair] = pairs.get(pair, 0) + freq

        if not pairs:
            break

        # Find the most common pair
        best_pair = max(pairs, key=pairs.get)
        merges.append(best_pair)

        print(f"Step {step + 1}: Merge '{best_pair[0]}' + '{best_pair[1]}' "
              f"→ '{best_pair[0]}{best_pair[1]}' "
              f"(appeared {pairs[best_pair]} times)")

        # Apply the merge: replace all occurrences of this pair
        new_words = {}
        for word, freq in words.items():
            new_word = word.replace(
                f"{best_pair[0]} {best_pair[1]}",
                f"{best_pair[0]}{best_pair[1]}"
            )
            new_words[new_word] = freq
        words = new_words

        # Show current state of words
        print(f"  Words now: {words}\n")

    return merges, words


# ============================================================
# DEMO: Train BPE on a tiny corpus
# ============================================================
corpus = "low low low low low lowest lowest newer newer newer wider"

print("=" * 60)
print("BPE Training Demo")
print(f"Corpus: \"{corpus}\"")
print("=" * 60 + "\n")

merges, final_words = simple_bpe_train(corpus, num_merges=10)

print("=" * 60)
print("Final merge rules learned:")
for i, (a, b) in enumerate(merges):
    print(f"  Rule {i+1}: '{a}' + '{b}' → '{a}{b}'")

# ============================================================
# HOW BPE TOKENIZES NEW TEXT
# ============================================================
print("""
=== How BPE Tokenizes New Text ===

After training, to tokenize a new word like "lowest":
  1. Start with characters: ['l', 'o', 'w', 'e', 's', 't']
  2. Apply merge rules in order:
     - Rule 'l'+'o' → 'lo':   ['lo', 'w', 'e', 's', 't']
     - Rule 'lo'+'w' → 'low': ['low', 'e', 's', 't']
     - Rule 'e'+'s' → 'es':   ['low', 'es', 't']
     - Rule 'es'+'t' → 'est': ['low', 'est']
  3. Final tokens: ['low', 'est']

For a RARE word like "xylophone":
  Characters are mostly left as-is because no merge rules apply.
  → ['x', 'y', 'l', 'o', 'p', 'h', 'o', 'n', 'e']

This is the beauty of BPE:
  - COMMON words/subwords become single tokens → efficient
  - RARE words get broken into characters → still works
  - Vocabulary size is controllable (= number of merges + base chars)

GPT-2 uses 50,000 merges → vocab size of ~50,257
microGPT uses 0 merges → vocab is just individual characters (~27)
""")
