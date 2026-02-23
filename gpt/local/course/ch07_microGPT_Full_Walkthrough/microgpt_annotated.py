"""
Chapter 07: microGPT — Fully Annotated

This is the COMPLETE microgpt.py by @karpathy, with extensive
annotations for CS students. Every section maps back to the
concepts from Chapters 01-06.

Original: ../../8627fe009c40f57531cb18360106ce95/microgpt.py

To run: python microgpt_annotated.py
(Takes several minutes — it's pure Python, no GPU!)
You can reduce num_steps below to speed it up.
"""

# =====================================================================
# SECTION 1: IMPORTS & SETUP (no external dependencies!)
# =====================================================================
# This entire GPT uses ONLY Python standard library.
# "Everything else is just efficiency." — @karpathy

import os       # os.path.exists
import math     # math.log, math.exp
import random   # random.seed, random.choices, random.gauss, random.shuffle
random.seed(42) # Fixed seed for reproducibility

# =====================================================================
# SECTION 2: DATASET (Chapter 01 — The Data)
# =====================================================================
# Downloads a list of ~32,000 baby names.
# Each name becomes a "document" that the model learns to generate.

if not os.path.exists('input.txt'):
    import urllib.request
    names_url = 'https://raw.githubusercontent.com/karpathy/makemore/988aa59/names.txt'
    urllib.request.urlretrieve(names_url, 'input.txt')
docs = [line.strip() for line in open('input.txt') if line.strip()]
random.shuffle(docs)
print(f"num docs: {len(docs)}")
print(f"first 5 names: {docs[:5]}")

# =====================================================================
# SECTION 3: TOKENIZER (Chapter 02 — Tokenization)
# =====================================================================
# Character-level tokenizer: each unique letter → integer
# Plus one special BOS (Beginning of Sequence) token

uchars = sorted(set(''.join(docs)))  # unique characters: ['a', 'b', ..., 'z']
BOS = len(uchars)                     # BOS gets the next available ID (26)
vocab_size = len(uchars) + 1          # 27 total tokens
print(f"vocab size: {vocab_size}")
print(f"characters: {uchars}")
print(f"BOS token id: {BOS}")

# Example: tokenizing "emma"
example = "emma"
example_tokens = [BOS] + [uchars.index(ch) for ch in example] + [BOS]
print(f"'{example}' tokenized: {example_tokens}")

# =====================================================================
# SECTION 4: AUTOGRAD ENGINE (Chapter 03 — The Value Class)
# =====================================================================
# Every number in the model is a Value object that tracks:
#   - Its numeric value (.data)
#   - Its gradient (.grad) — filled in during backward pass
#   - How it was computed (_children, _local_grads) — for chain rule

class Value:
    __slots__ = ('data', 'grad', '_children', '_local_grads')

    def __init__(self, data, children=(), local_grads=()):
        self.data = data
        self.grad = 0
        self._children = children
        self._local_grads = local_grads

    # --- ARITHMETIC OPERATIONS ---
    # Each operation records the local derivative for the chain rule.

    def __add__(self, other):
        # d(a+b)/da = 1, d(a+b)/db = 1
        other = other if isinstance(other, Value) else Value(other)
        return Value(self.data + other.data, (self, other), (1, 1))

    def __mul__(self, other):
        # d(a*b)/da = b, d(a*b)/db = a
        other = other if isinstance(other, Value) else Value(other)
        return Value(self.data * other.data, (self, other), (other.data, self.data))

    def __pow__(self, other):
        # d(a^n)/da = n * a^(n-1)
        return Value(self.data**other, (self,), (other * self.data**(other-1),))

    def log(self):
        # d(log(a))/da = 1/a
        return Value(math.log(self.data), (self,), (1/self.data,))

    def exp(self):
        # d(exp(a))/da = exp(a)
        return Value(math.exp(self.data), (self,), (math.exp(self.data),))

    def relu(self):
        # d(relu(a))/da = 1 if a > 0, else 0
        return Value(max(0, self.data), (self,), (float(self.data > 0),))

    # Convenience methods for operator overloading
    def __neg__(self): return self * -1
    def __radd__(self, other): return self + other
    def __sub__(self, other): return self + (-other)
    def __rsub__(self, other): return other + (-self)
    def __rmul__(self, other): return self * other
    def __truediv__(self, other): return self * other**-1
    def __rtruediv__(self, other): return other * self**-1

    def backward(self):
        """Backpropagation: compute gradients via reverse-mode autodiff."""
        # Step 1: Topological sort (ensure children processed before parents)
        topo = []
        visited = set()
        def build_topo(v):
            if v not in visited:
                visited.add(v)
                for child in v._children:
                    build_topo(child)
                topo.append(v)
        build_topo(self)

        # Step 2: Propagate gradients backward through the graph
        self.grad = 1  # d(loss)/d(loss) = 1
        for v in reversed(topo):
            for child, local_grad in zip(v._children, v._local_grads):
                # Chain rule: child.grad += local_derivative * parent.grad
                child.grad += local_grad * v.grad

# =====================================================================
# SECTION 5: PARAMETER INITIALIZATION (Chapter 04 — Weight Matrices)
# =====================================================================
# The "knowledge" of the model lives in these weight matrices.
# Before training, they're random. After training, they encode patterns.

n_layer = 1      # 1 transformer block (GPT-2 uses 12-48)
n_embd = 16      # embedding dimension (GPT-2 uses 768-1600)
block_size = 16   # max context length (longest name is 15 chars)
n_head = 4        # attention heads (GPT-2 uses 12-25)
head_dim = n_embd // n_head  # = 4 dims per head

# Helper: create a matrix of Value objects initialized with small random values
matrix = lambda nout, nin, std=0.08: [
    [Value(random.gauss(0, std)) for _ in range(nin)]
    for _ in range(nout)
]

# The state_dict: all learnable parameters organized by name
state_dict = {
    'wte': matrix(vocab_size, n_embd),    # Token embeddings: 27 tokens × 16 dims
    'wpe': matrix(block_size, n_embd),    # Position embeddings: 16 positions × 16 dims
    'lm_head': matrix(vocab_size, n_embd) # Output layer: maps embeddings → token scores
}

# Each transformer layer has attention and MLP weights
for i in range(n_layer):
    state_dict[f'layer{i}.attn_wq'] = matrix(n_embd, n_embd)  # Query projection
    state_dict[f'layer{i}.attn_wk'] = matrix(n_embd, n_embd)  # Key projection
    state_dict[f'layer{i}.attn_wv'] = matrix(n_embd, n_embd)  # Value projection
    state_dict[f'layer{i}.attn_wo'] = matrix(n_embd, n_embd)  # Output projection
    state_dict[f'layer{i}.mlp_fc1'] = matrix(4 * n_embd, n_embd)  # MLP expand (16→64)
    state_dict[f'layer{i}.mlp_fc2'] = matrix(n_embd, 4 * n_embd)  # MLP compress (64→16)

# Flatten all parameters into one list (for the optimizer)
params = [p for mat in state_dict.values() for row in mat for p in row]
print(f"num params: {len(params)}")

# =====================================================================
# SECTION 6: MODEL ARCHITECTURE (Chapter 05 — Attention + MLP)
# =====================================================================
# The GPT model: token → embedding → transformer blocks → logits
# Follows GPT-2 with minor changes: RMSNorm instead of LayerNorm,
# no biases, ReLU instead of GELU.

def linear(x, w):
    """Linear layer: matrix-vector multiply (Chapter 04)"""
    return [sum(wi * xi for wi, xi in zip(wo, x)) for wo in w]

def softmax(logits):
    """Softmax: scores → probabilities (Chapter 04)"""
    max_val = max(val.data for val in logits)
    exps = [(val - max_val).exp() for val in logits]
    total = sum(exps)
    return [e / total for e in exps]

def rmsnorm(x):
    """RMSNorm: stabilize values (Chapter 04)"""
    ms = sum(xi * xi for xi in x) / len(x)
    scale = (ms + 1e-5) ** -0.5
    return [xi * scale for xi in x]

def gpt(token_id, pos_id, keys, values):
    """
    Process one token through the GPT model.
    
    This is called once per token position during both training and inference.
    Uses KV-cache: stores keys/values so previous positions aren't recomputed.
    
    Args:
        token_id: integer, which token (0-26)
        pos_id: integer, position in sequence (0-15)
        keys: list of lists, accumulated key vectors per layer
        values: list of lists, accumulated value vectors per layer
    
    Returns:
        logits: list of 27 Values, scores for each possible next token
    """
    # STEP 1: Embedding lookup (Chapter 02)
    # Each token gets a learned vector + a position vector
    tok_emb = state_dict['wte'][token_id]   # [n_embd] = [16]
    pos_emb = state_dict['wpe'][pos_id]     # [n_embd] = [16]
    x = [t + p for t, p in zip(tok_emb, pos_emb)]  # combine
    x = rmsnorm(x)  # normalize

    # STEP 2: Transformer blocks (Chapter 05)
    for li in range(n_layer):

        # --- ATTENTION BLOCK ---
        x_residual = x
        x = rmsnorm(x)

        # Project to Q, K, V
        q = linear(x, state_dict[f'layer{li}.attn_wq'])
        k = linear(x, state_dict[f'layer{li}.attn_wk'])
        v = linear(x, state_dict[f'layer{li}.attn_wv'])

        # Store K, V in cache for future tokens
        keys[li].append(k)
        values[li].append(v)

        # Multi-head attention
        x_attn = []
        for h in range(n_head):
            hs = h * head_dim  # start index for this head's slice

            # Get this head's Q, K, V
            q_h = q[hs:hs+head_dim]
            k_h = [ki[hs:hs+head_dim] for ki in keys[li]]   # all cached keys
            v_h = [vi[hs:hs+head_dim] for vi in values[li]]  # all cached values

            # Attention scores: Q · K / sqrt(d)
            attn_logits = [
                sum(q_h[j] * k_h[t][j] for j in range(head_dim)) / head_dim**0.5
                for t in range(len(k_h))
            ]

            # Attention weights (softmax)
            attn_weights = softmax(attn_logits)

            # Weighted sum of values
            head_out = [
                sum(attn_weights[t] * v_h[t][j] for t in range(len(v_h)))
                for j in range(head_dim)
            ]
            x_attn.extend(head_out)  # concatenate heads

        # Output projection + residual connection
        x = linear(x_attn, state_dict[f'layer{li}.attn_wo'])
        x = [a + b for a, b in zip(x, x_residual)]

        # --- MLP BLOCK ---
        x_residual = x
        x = rmsnorm(x)
        x = linear(x, state_dict[f'layer{li}.mlp_fc1'])  # expand: 16 → 64
        x = [xi.relu() for xi in x]                       # activate
        x = linear(x, state_dict[f'layer{li}.mlp_fc2'])  # compress: 64 → 16
        x = [a + b for a, b in zip(x, x_residual)]       # residual

    # STEP 3: Output head (map embedding → token scores)
    logits = linear(x, state_dict['lm_head'])  # [vocab_size] = [27]
    return logits

# =====================================================================
# SECTION 7: ADAM OPTIMIZER SETUP (Chapter 06)
# =====================================================================
learning_rate, beta1, beta2, eps_adam = 0.01, 0.85, 0.99, 1e-8
m = [0.0] * len(params)  # momentum buffer
v = [0.0] * len(params)  # velocity buffer

# =====================================================================
# SECTION 8: TRAINING LOOP (Chapter 06)
# =====================================================================
# For each step:
#   1. Take a name, tokenize it
#   2. Forward pass: predict each next character
#   3. Compute cross-entropy loss
#   4. Backward pass: compute gradients
#   5. Adam update: adjust parameters

num_steps = 200  # Reduced from 1000 for faster demo. Increase for better results!
print(f"\nTraining for {num_steps} steps...")
print("(Reduce num_steps in the code if this is too slow)\n")

for step in range(num_steps):

    # 1. Pick a document (name) and tokenize it
    doc = docs[step % len(docs)]
    tokens = [BOS] + [uchars.index(ch) for ch in doc] + [BOS]
    n = min(block_size, len(tokens) - 1)

    # 2-3. Forward pass: predict next token at each position, accumulate loss
    keys_cache = [[] for _ in range(n_layer)]
    values_cache = [[] for _ in range(n_layer)]
    losses = []

    for pos_id in range(n):
        token_id = tokens[pos_id]
        target_id = tokens[pos_id + 1]

        # Forward: get logits for this position
        logits = gpt(token_id, pos_id, keys_cache, values_cache)

        # Softmax → probabilities
        probs = softmax(logits)

        # Cross-entropy loss for this position
        loss_t = -probs[target_id].log()
        losses.append(loss_t)

    # Average loss over all positions in this name
    loss = (1 / n) * sum(losses)

    # 4. Backward pass: compute all gradients
    loss.backward()

    # 5. Adam optimizer: update all parameters
    lr_t = learning_rate * (1 - step / num_steps)  # linear LR decay
    for i, p in enumerate(params):
        m[i] = beta1 * m[i] + (1 - beta1) * p.grad
        v[i] = beta2 * v[i] + (1 - beta2) * p.grad ** 2
        m_hat = m[i] / (1 - beta1 ** (step + 1))
        v_hat = v[i] / (1 - beta2 ** (step + 1))
        p.data -= lr_t * m_hat / (v_hat ** 0.5 + eps_adam)
        p.grad = 0  # reset for next step

    if step % 20 == 0 or step == num_steps - 1:
        print(f"step {step+1:4d} / {num_steps:4d} | loss {loss.data:.4f}")

# =====================================================================
# SECTION 9: INFERENCE (Chapter 01 — Generating Text)
# =====================================================================
# Now the model generates new names that "sound like" the training data.
# Process: start with BOS, predict next token, repeat until BOS again.

temperature = 0.5  # lower = more conservative, higher = more creative

print("\n--- inference (new, hallucinated names) ---")
for sample_idx in range(20):
    keys_cache = [[] for _ in range(n_layer)]
    values_cache = [[] for _ in range(n_layer)]
    token_id = BOS
    sample = []

    for pos_id in range(block_size):
        logits = gpt(token_id, pos_id, keys_cache, values_cache)
        # Apply temperature: divide logits before softmax
        probs = softmax([l / temperature for l in logits])
        # Sample from the probability distribution
        token_id = random.choices(range(vocab_size),
                                  weights=[p.data for p in probs])[0]
        if token_id == BOS:
            break  # end of name
        sample.append(uchars[token_id])

    print(f"sample {sample_idx+1:2d}: {''.join(sample)}")

print("""
=== That's the entire microGPT! ===

200 lines that implement:
  ✓ Dataset loading and tokenization
  ✓ Automatic differentiation (autograd)
  ✓ Transformer architecture (attention + MLP)
  ✓ Training with Adam optimizer
  ✓ Text generation (inference)

Next: Chapter 08 shows how PyTorch makes this 100x faster and cleaner.
""")
