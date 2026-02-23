"""
Chapter 11: microGPT vs minGPT — Detailed Comparison

This script walks through each section of both implementations,
showing the equivalent code side-by-side with explanations.

No dependencies required — this is a reading/reference script.
"""

# ============================================================
# Helper to format code snippets
# ============================================================
def compare(title, micro_code, min_code, explanation):
    print("=" * 70)
    print(f"  {title}")
    print("=" * 70)
    print(f"\n  microGPT (pure Python):")
    for line in micro_code.strip().split('\n'):
        print(f"    {line}")
    print(f"\n  minGPT (PyTorch):")
    for line in min_code.strip().split('\n'):
        print(f"    {line}")
    print(f"\n  → {explanation}\n")


# ============================================================
# COMPARISON 1: Tokenization
# ============================================================
compare(
    "1. TOKENIZATION",
    """
uchars = sorted(set(''.join(docs)))
BOS = len(uchars)
vocab_size = len(uchars) + 1
tokens = [BOS] + [uchars.index(ch) for ch in doc] + [BOS]
""",
    """
# CharDataset (projects/chargpt/chargpt.py)
chars = sorted(list(set(data)))
self.stoi = {ch:i for i,ch in enumerate(chars)}
self.itos = {i:ch for i,ch in enumerate(chars)}
dix = [self.stoi[s] for s in chunk]
x = torch.tensor(dix[:-1], dtype=torch.long)
""",
    "Same idea: map characters to integers. minGPT wraps it in a Dataset class\n"
    "  and returns PyTorch tensors instead of plain lists."
)

# ============================================================
# COMPARISON 2: Embeddings
# ============================================================
compare(
    "2. EMBEDDINGS",
    """
# Token + Position embedding (inline in gpt function)
tok_emb = state_dict['wte'][token_id]    # list lookup
pos_emb = state_dict['wpe'][pos_id]      # list lookup
x = [t + p for t, p in zip(tok_emb, pos_emb)]
""",
    """
# GPT.__init__:
self.transformer.wte = nn.Embedding(vocab_size, n_embd)
self.transformer.wpe = nn.Embedding(block_size, n_embd)

# GPT.forward:
tok_emb = self.transformer.wte(idx)    # (B, T, n_embd)
pos_emb = self.transformer.wpe(pos)    # (1, T, n_embd)
x = tok_emb + pos_emb                 # broadcasts!
""",
    "microGPT looks up one embedding at a time (one token, one position).\n"
    "  minGPT looks up ALL embeddings for ALL tokens in the batch at once."
)

# ============================================================
# COMPARISON 3: Attention
# ============================================================
compare(
    "3. MULTI-HEAD ATTENTION",
    """
# Process one token at a time, accumulate KV cache
q = linear(x, state_dict[f'layer{li}.attn_wq'])
k = linear(x, state_dict[f'layer{li}.attn_wk'])
v = linear(x, state_dict[f'layer{li}.attn_wv'])
keys[li].append(k)
values[li].append(v)

for h in range(n_head):
    hs = h * head_dim
    q_h = q[hs:hs+head_dim]
    k_h = [ki[hs:hs+head_dim] for ki in keys[li]]
    v_h = [vi[hs:hs+head_dim] for vi in values[li]]
    attn_logits = [sum(...) / head_dim**0.5 for t in ...]
    attn_weights = softmax(attn_logits)
    head_out = [sum(...) for j in range(head_dim)]
    x_attn.extend(head_out)
""",
    """
# Process ALL tokens at once with masked attention
q, k, v = self.c_attn(x).split(self.n_embd, dim=2)
k = k.view(B, T, nh, hs).transpose(1, 2)  # (B, nh, T, hs)
q = q.view(B, T, nh, hs).transpose(1, 2)
v = v.view(B, T, nh, hs).transpose(1, 2)

att = (q @ k.transpose(-2, -1)) * (1.0 / sqrt(hs))
att = att.masked_fill(self.bias[:,:,:T,:T] == 0, -inf)
att = F.softmax(att, dim=-1)
y = att @ v
y = y.transpose(1, 2).contiguous().view(B, T, C)
""",
    "This is the BIGGEST difference between the two implementations:\n"
    "  - microGPT: sequential (one token at a time, Python loops over heads)\n"
    "  - minGPT: parallel (all tokens, all heads, all batches in one matmul)\n"
    "  Same math, but minGPT is orders of magnitude faster."
)

# ============================================================
# COMPARISON 4: MLP Block
# ============================================================
compare(
    "4. MLP (FEEDFORWARD) BLOCK",
    """
x = linear(x, state_dict[f'layer{li}.mlp_fc1'])  # expand
x = [xi.relu() for xi in x]                       # activate
x = linear(x, state_dict[f'layer{li}.mlp_fc2'])  # compress
""",
    """
# Block.__init__:
self.mlp = nn.ModuleDict(dict(
    c_fc   = nn.Linear(n_embd, 4 * n_embd),
    c_proj = nn.Linear(4 * n_embd, n_embd),
    act    = NewGELU(),
))

# Block.forward:
self.mlpf = lambda x: m.dropout(m.c_proj(m.act(m.c_fc(x))))
""",
    "Identical structure: expand → activate → compress.\n"
    "  microGPT uses ReLU, minGPT uses GELU (smoother).\n"
    "  minGPT adds dropout for regularization."
)

# ============================================================
# COMPARISON 5: Normalization
# ============================================================
compare(
    "5. NORMALIZATION",
    """
def rmsnorm(x):
    ms = sum(xi * xi for xi in x) / len(x)
    scale = (ms + 1e-5) ** -0.5
    return [xi * scale for xi in x]
""",
    """
# Uses PyTorch's built-in LayerNorm
self.ln_1 = nn.LayerNorm(n_embd)
self.ln_2 = nn.LayerNorm(n_embd)

# In forward:
x = x + self.attn(self.ln_1(x))
""",
    "RMSNorm (microGPT) only scales by magnitude.\n"
    "  LayerNorm (minGPT) also shifts by mean and has learnable parameters.\n"
    "  Both serve the same purpose: keep values in a stable range."
)

# ============================================================
# COMPARISON 6: Residual Connections
# ============================================================
compare(
    "6. RESIDUAL CONNECTIONS",
    """
x_residual = x
x = rmsnorm(x)
# ... attention ...
x = [a + b for a, b in zip(x, x_residual)]  # ADD residual

x_residual = x
x = rmsnorm(x)
# ... MLP ...
x = [a + b for a, b in zip(x, x_residual)]  # ADD residual
""",
    """
def forward(self, x):
    x = x + self.attn(self.ln_1(x))    # residual around attention
    x = x + self.mlpf(self.ln_2(x))    # residual around MLP
    return x
""",
    "Identical concept: output = input + transformation(input).\n"
    "  minGPT is more concise because PyTorch handles the element-wise add."
)

# ============================================================
# COMPARISON 7: Training Loop
# ============================================================
compare(
    "7. TRAINING LOOP",
    """
for step in range(num_steps):
    doc = docs[step % len(docs)]
    tokens = [BOS] + [uchars.index(ch) for ch in doc] + [BOS]
    # ... forward each token through gpt() ...
    loss = (1/n) * sum(losses)
    loss.backward()
    # Manual Adam:
    for i, p in enumerate(params):
        m[i] = beta1 * m[i] + (1-beta1) * p.grad
        v[i] = beta2 * v[i] + (1-beta2) * p.grad**2
        # ... bias correction, update ...
        p.data -= lr_t * m_hat / (v_hat**0.5 + eps)
        p.grad = 0
""",
    """
while True:
    batch = next(data_iter)
    x, y = [t.to(self.device) for t in batch]
    logits, self.loss = model(x, y)
    model.zero_grad(set_to_none=True)
    self.loss.backward()
    clip_grad_norm_(model.parameters(), config.grad_norm_clip)
    self.optimizer.step()
""",
    "Same loop: forward → loss → backward → update.\n"
    "  microGPT: 1 document at a time, manual Adam, ~15 lines.\n"
    "  minGPT: batched, built-in optimizer, gradient clipping, ~10 lines."
)

# ============================================================
# COMPARISON 8: Inference
# ============================================================
compare(
    "8. TEXT GENERATION (INFERENCE)",
    """
token_id = BOS
for pos_id in range(block_size):
    logits = gpt(token_id, pos_id, keys, values)
    probs = softmax([l / temperature for l in logits])
    token_id = random.choices(range(vocab_size),
                              weights=[p.data for p in probs])[0]
    if token_id == BOS: break
    sample.append(uchars[token_id])
""",
    """
for _ in range(max_new_tokens):
    idx_cond = idx if idx.size(1) <= self.block_size \\
               else idx[:, -self.block_size:]
    logits, _ = self(idx_cond)
    logits = logits[:, -1, :] / temperature
    if top_k is not None:
        v, _ = torch.topk(logits, top_k)
        logits[logits < v[:, [-1]]] = -float('Inf')
    probs = F.softmax(logits, dim=-1)
    if do_sample:
        idx_next = torch.multinomial(probs, num_samples=1)
    else:
        _, idx_next = torch.topk(probs, k=1, dim=-1)
    idx = torch.cat((idx, idx_next), dim=1)
""",
    "microGPT: sequential, one token at a time, KV-cache.\n"
    "  minGPT: can process batches, supports top-k filtering, greedy mode.\n"
    "  Both: predict one token, append, repeat."
)

# ============================================================
# SUMMARY TABLE
# ============================================================
print("=" * 70)
print("  SUMMARY: When to Use Which")
print("=" * 70)
print("""
  ┌──────────────────────────────────────────────────────────────────┐
  │  USE microGPT WHEN YOU WANT TO:                                  │
  │    • Understand exactly how every piece of GPT works             │
  │    • See autograd, attention, training loop without any magic     │
  │    • Learn the algorithm, not the framework                      │
  │    • Have a single-file reference with zero dependencies         │
  │                                                                  │
  │  USE minGPT WHEN YOU WANT TO:                                    │
  │    • Actually train on real data (Shakespeare, code, etc.)       │
  │    • Experiment with different model sizes                       │
  │    • Use GPU acceleration                                        │
  │    • Load pretrained GPT-2 weights                               │
  │    • Build projects on top of a clean GPT implementation         │
  │                                                                  │
  │  THE KEY INSIGHT:                                                │
  │    microGPT IS minGPT, just without the efficiency tricks.       │
  │    If you understand microGPT, you understand minGPT.            │
  │    If you understand minGPT, you understand GPT-2.               │
  │    The architecture is the same at every scale.                  │
  └──────────────────────────────────────────────────────────────────┘
""")
