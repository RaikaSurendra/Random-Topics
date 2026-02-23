# Chapter 01: What is a Language Model?

## The One-Sentence Summary

A language model is a program that **predicts the next word** (or character) given some previous words.

---

## Think of It Like Autocomplete

You've used autocomplete on your phone:

```
You type: "How are ___"
Phone suggests: "you"
```

That's a language model! It looked at "How are" and predicted "you" is the most likely next word.

GPT is the same idea — just way more powerful.

---

## The Core Loop

Every language model follows this pattern:

```
1. Look at some text               → "The cat sat on the"
2. Predict the next word            → "mat" (70%), "floor" (20%), "dog" (10%)
3. Pick one (sample or take best)   → "mat"
4. Append it and repeat             → "The cat sat on the mat"
```

That's it. GPT generates entire essays by repeating steps 1-4 over and over.

---

## But How Does It Know?

The model **learns patterns from data**. If you show it millions of sentences, it notices:
- "The cat sat on the ___" → usually "mat", "floor", "chair"
- "Once upon a ___" → usually "time"
- "def __init__(self, ___" → usually a parameter name

It doesn't "understand" language. It's really good at **pattern matching**.

---

## The Two Phases

### Phase 1: Training (Learning)
- Feed the model tons of text
- For each position, it tries to predict the next character/word
- When it's wrong, adjust its internal numbers to be less wrong next time
- Repeat millions of times

### Phase 2: Inference (Generating)
- Give it a starting text (prompt)
- Let it predict the next token, one at a time
- It generates new text that "sounds like" its training data

---

## Characters vs Words vs Tokens

Language models can predict at different levels:

| Level | Example Input | Predicts |
|---|---|---|
| **Character-level** | `['H', 'e', 'l', 'l']` | `'o'` |
| **Word-level** | `['The', 'cat']` | `'sat'` |
| **Subword (BPE)** | `['The', ' cat', ' s']` | `'at'` |

- **microGPT** uses character-level (simplest)
- **minGPT** can use any level, but demos use character-level and BPE

---

## What's Inside a Language Model?

At its core, a language model is just a **function with adjustable numbers** (parameters):

```
f(input_tokens, parameters) → probability of each possible next token
```

- The **parameters** are millions of numbers that encode patterns
- **Training** = finding good values for those numbers
- **Architecture** = how those numbers are organized and combined

GPT uses an architecture called a **Transformer**, which we'll learn about in Chapter 05.

---

## Run the Example

The file `language_model_idea.py` in this folder shows the simplest possible "language model" — just counting letter frequencies. It's silly and bad, but it captures the core idea.

```bash
python language_model_idea.py
```
