# Chapter 12: Exercises & Next Steps

## Congratulations!

You now understand the complete GPT architecture — from raw text to generated output. Here are exercises to solidify your understanding, organized by difficulty.

---

## Beginner Exercises (Chapters 01-04)

### Exercise 1: Bigram Language Model
Build a character-level bigram model (like Ch01) but train it using gradient descent and the Value class (Ch03) instead of counting.

### Exercise 2: Custom Tokenizer
Modify the character tokenizer from Ch02 to handle uppercase letters, spaces, and punctuation. Test it on a paragraph of text.

### Exercise 3: Autograd Extensions
Add `tanh()` and `sigmoid()` operations to the Value class. Verify their gradients by comparing with numerical differentiation.

### Exercise 4: Softmax Temperature Explorer
Write a script that visualizes how different temperatures (0.1 to 5.0) affect the output distribution for a fixed set of logits.

---

## Intermediate Exercises (Chapters 05-07)

### Exercise 5: Attention Visualization
Modify the attention code from Ch05 to print attention weights for each head. Feed in a known pattern (e.g., "abab") and see which positions attend to which.

### Exercise 6: microGPT Hyperparameter Tuning
Modify microGPT's hyperparameters and measure the effect on loss:
- Try `n_head = 1, 2, 4, 8`
- Try `n_embd = 8, 16, 32`
- Try `n_layer = 1, 2, 3`

### Exercise 7: Different Dataset
Modify microGPT to train on a different character-level dataset (e.g., country names, short words, or DNA sequences like "ATCGATCG").

---

## Advanced Exercises (Chapters 08-11)

### Exercise 8: minGPT Character Generator
Use minGPT's `chargpt` project to train on a text file of your choice. Try song lyrics, code, or recipes.

### Exercise 9: Add Dropout to microGPT
Implement dropout in pure Python and add it to microGPT's attention and MLP blocks. Does it help with overfitting?

### Exercise 10: Implement Beam Search
Instead of sampling one token at a time, implement beam search for text generation: maintain the top-k most likely sequences at each step.

### Exercise 11: Weight Sharing
In real GPT-2, the token embedding matrix (`wte`) and the output head (`lm_head`) share weights. Implement this in either microGPT or the simplified minGPT from Ch09.

---

## Challenge Exercises

### Exercise 12: Build nanoGPT
Combine the best of both implementations: write a GPT that uses PyTorch but is contained in a single file under 300 lines. (This is essentially what Karpathy did with nanoGPT!)

### Exercise 13: Positional Encoding Variants
Replace learned positional embeddings with sinusoidal positional encodings (from the original "Attention is All You Need" paper). Compare results.

---

## Next Steps & Further Reading

### Karpathy's Other Projects
- **nanoGPT**: The successor to minGPT — more practical, reproduces benchmarks
  - https://github.com/karpathy/nanoGPT
- **makemore**: Character-level name generation (the dataset microGPT uses)
  - https://github.com/karpathy/makemore
- **Karpathy's YouTube**: Full video lectures building these from scratch
  - https://www.youtube.com/c/AndrejKarpathy

### Papers to Read
1. **"Attention Is All You Need"** (2017) — The original transformer paper
2. **"Improving Language Understanding by Generative Pre-Training"** (GPT-1, 2018)
3. **"Language Models are Few-Shot Learners"** (GPT-3, 2020)

### Topics to Explore Next
- **Tokenization deep-dive**: SentencePiece, tiktoken
- **Training at scale**: Distributed training, mixed precision
- **Fine-tuning**: LoRA, RLHF, instruction tuning
- **Inference optimization**: KV-cache, quantization, speculative decoding
- **Other architectures**: BERT (encoder), T5 (encoder-decoder)

---

## Quick Reference Card

```
GPT in one paragraph:
  Tokenize text into integers. Embed each token into a vector.
  Add position embeddings. Pass through N transformer blocks,
  each doing: normalize → multi-head attention → residual →
  normalize → MLP → residual. Final linear layer maps to
  vocab-sized logits. Softmax gives next-token probabilities.
  Train with cross-entropy loss and Adam optimizer.
  Generate by sampling one token at a time.
```
