# Understanding GPT From Scratch

## A Course Based on Andrej Karpathy's Implementations

**Target Audience:** Early CS students who understand basic programming (Python, loops, functions, classes) but have no background in AI/ML or GPT.

---

## What This Course Covers

This course walks you through **two real GPT implementations** by Andrej Karpathy, starting from zero AI knowledge:

| Implementation | Location | What It Is |
|---|---|---|
| **microGPT** | `../../8627fe009c40f57531cb18360106ce95/microgpt.py` | ~200 lines of pure Python. No libraries. Builds everything from scratch — autograd, neural networks, the full transformer. |
| **minGPT** | `../../minGPT/` | A clean PyTorch implementation. Production-style code with proper modules, training infrastructure, and real GPT-2 compatibility. |

---

## Course Structure

| Folder | Title | Key Idea |
|---|---|---|
| `ch01_What_is_a_Language_Model/` | What is a Language Model? | The big picture — predicting the next word |
| `ch02_Tokenization/` | Tokenization | Turning text into numbers a computer can process |
| `ch03_Autograd/` | Autograd | Teaching computers to do calculus automatically |
| `ch04_Neural_Network_Building_Blocks/` | Neural Network Building Blocks | Linear layers, activation functions, softmax |
| `ch05_Attention_and_Transformers/` | Attention & Transformers | The core innovation behind GPT |
| `ch06_Training_Loop_and_Optimization/` | Training: How Models Learn | Loss functions, backprop, optimizers |
| `ch07_microGPT_Full_Walkthrough/` | microGPT: Full Walkthrough | Line-by-line through the 200-line pure-Python GPT |
| `ch08_Scaling_Up_with_PyTorch/` | Scaling Up with PyTorch | Why we need frameworks, intro to PyTorch |
| `ch09_minGPT_Model_Deep_Dive/` | minGPT: Model Deep Dive | The production-quality GPT architecture |
| `ch10_Training_and_Inference/` | minGPT: Training & Inference | Training loops, text generation, real demos |
| `ch11_Side_by_Side_Comparison/` | Side-by-Side Comparison | microGPT vs minGPT — same ideas, different scales |
| `ch12_Exercises_and_Next_Steps/` | Exercises & Next Steps | Hands-on challenges and further reading |

---

## How to Use This Course

1. **Read chapters in order** — each builds on the previous one
2. **Run the code examples** — every chapter has runnable `.py` files in its folder
3. **Chapters 01-06** teach the concepts with small, isolated examples
4. **Chapters 07-10** apply those concepts to the real Karpathy code
5. **Chapter 11** ties everything together
6. **Chapter 12** gives you challenges to test your understanding

## Prerequisites

- Python basics: variables, loops, functions, classes, lists, dictionaries
- Basic math: addition, multiplication, exponents (no calculus needed — we teach it!)
- A terminal / command line
- Python 3.8+ installed

## Running Examples

```bash
# For chapters 01-07 (pure Python, no dependencies):
python ch01_What_is_a_Language_Model/language_model_idea.py

# For chapters 08-10 (needs PyTorch):
pip install torch
python ch08_Scaling_Up_with_PyTorch/pytorch_basics.py
```
