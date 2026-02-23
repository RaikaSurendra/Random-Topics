"""
Chapter 10: minGPT Trainer — Annotated

This builds a simplified version of minGPT's Trainer and trains
a tiny GPT on a sorting task (from demo.ipynb).

Source: ../../minGPT/mingpt/trainer.py
Demo:   ../../minGPT/demo.ipynb

Requires: pip install torch
"""

try:
    import torch
    import torch.nn as nn
    from torch.nn import functional as F
    from torch.utils.data import Dataset, DataLoader
    import math
    import time
except ImportError:
    print("PyTorch not installed. Run: pip install torch")
    exit(1)

torch.manual_seed(3407)

# ============================================================
# MODEL: Simplified GPT (from Chapter 09)
# ============================================================
class NewGELU(nn.Module):
    def forward(self, x):
        return 0.5 * x * (1.0 + torch.tanh(math.sqrt(2.0/math.pi) * (x + 0.044715 * torch.pow(x, 3.0))))

class CausalSelfAttention(nn.Module):
    def __init__(self, n_embd, n_head, block_size):
        super().__init__()
        self.c_attn = nn.Linear(n_embd, 3 * n_embd)
        self.c_proj = nn.Linear(n_embd, n_embd)
        self.register_buffer("bias", torch.tril(torch.ones(block_size, block_size)).view(1,1,block_size,block_size))
        self.n_head = n_head
        self.n_embd = n_embd
    def forward(self, x):
        B, T, C = x.size()
        q, k, v = self.c_attn(x).split(self.n_embd, dim=2)
        k = k.view(B, T, self.n_head, C//self.n_head).transpose(1, 2)
        q = q.view(B, T, self.n_head, C//self.n_head).transpose(1, 2)
        v = v.view(B, T, self.n_head, C//self.n_head).transpose(1, 2)
        att = (q @ k.transpose(-2,-1)) * (1.0/math.sqrt(k.size(-1)))
        att = att.masked_fill(self.bias[:,:,:T,:T]==0, float('-inf'))
        att = F.softmax(att, dim=-1)
        y = att @ v
        y = y.transpose(1, 2).contiguous().view(B, T, C)
        return self.c_proj(y)

class Block(nn.Module):
    def __init__(self, n_embd, n_head, block_size):
        super().__init__()
        self.ln_1 = nn.LayerNorm(n_embd)
        self.attn = CausalSelfAttention(n_embd, n_head, block_size)
        self.ln_2 = nn.LayerNorm(n_embd)
        self.mlp = nn.Sequential(
            nn.Linear(n_embd, 4*n_embd), NewGELU(), nn.Linear(4*n_embd, n_embd))
    def forward(self, x):
        x = x + self.attn(self.ln_1(x))
        x = x + self.mlp(self.ln_2(x))
        return x

class GPT(nn.Module):
    def __init__(self, vocab_size, block_size, n_layer, n_head, n_embd):
        super().__init__()
        self.block_size = block_size
        self.wte = nn.Embedding(vocab_size, n_embd)
        self.wpe = nn.Embedding(block_size, n_embd)
        self.blocks = nn.ModuleList([Block(n_embd, n_head, block_size) for _ in range(n_layer)])
        self.ln_f = nn.LayerNorm(n_embd)
        self.head = nn.Linear(n_embd, vocab_size, bias=False)
        self.apply(self._init_weights)

    def _init_weights(self, module):
        if isinstance(module, nn.Linear):
            torch.nn.init.normal_(module.weight, mean=0.0, std=0.02)
            if module.bias is not None: torch.nn.init.zeros_(module.bias)
        elif isinstance(module, nn.Embedding):
            torch.nn.init.normal_(module.weight, mean=0.0, std=0.02)

    def forward(self, idx, targets=None):
        B, T = idx.size()
        pos = torch.arange(T, dtype=torch.long, device=idx.device)
        x = self.wte(idx) + self.wpe(pos)
        for block in self.blocks:
            x = block(x)
        logits = self.head(self.ln_f(x))
        loss = None
        if targets is not None:
            loss = F.cross_entropy(logits.view(-1, logits.size(-1)), targets.view(-1), ignore_index=-1)
        return logits, loss

    @torch.no_grad()
    def generate(self, idx, max_new_tokens):
        for _ in range(max_new_tokens):
            idx_cond = idx if idx.size(1) <= self.block_size else idx[:, -self.block_size:]
            logits, _ = self(idx_cond)
            logits = logits[:, -1, :]
            probs = F.softmax(logits, dim=-1)
            _, idx_next = torch.topk(probs, k=1, dim=-1)
            idx = torch.cat((idx, idx_next), dim=1)
        return idx

# ============================================================
# DATASET: Sorting task (from demo.ipynb)
# ============================================================
class SortDataset(Dataset):
    """
    Input:  [2, 0, 1]    → Output: [0, 1, 2]
    Concatenated: [2, 0, 1, 0, 1, 2]
    
    The model learns to "sort" by seeing thousands of examples.
    This proves GPT can learn algorithms, not just language!
    """
    def __init__(self, split, length=6, num_digits=3):
        self.split = split
        self.length = length
        self.num_digits = num_digits

    def __len__(self):
        return 10000

    def get_vocab_size(self):
        return self.num_digits

    def get_block_size(self):
        return self.length * 2 - 1

    def __getitem__(self, idx):
        import pickle
        while True:
            inp = torch.randint(self.num_digits, size=(self.length,), dtype=torch.long)
            h = hash(pickle.dumps(inp.tolist()))
            if (h % 4 == 0) == (self.split == 'test'):
                break
        sol = torch.sort(inp)[0]
        cat = torch.cat((inp, sol), dim=0)
        x = cat[:-1].clone()
        y = cat[1:].clone()
        y[:self.length - 1] = -1  # mask: don't predict at input positions
        return x, y

# ============================================================
# TRAINING LOOP (annotated version of mingpt/trainer.py)
# ============================================================
print("=" * 60)
print("Training GPT to Sort Numbers")
print("=" * 60)

# Create dataset and model
train_dataset = SortDataset('train')
test_dataset = SortDataset('test')

model = GPT(
    vocab_size=train_dataset.get_vocab_size(),  # 3 digits: 0, 1, 2
    block_size=train_dataset.get_block_size(),  # 11 positions
    n_layer=3, n_head=3, n_embd=48              # gpt-nano
)

n_params = sum(p.numel() for p in model.parameters())
print(f"\nModel: gpt-nano, {n_params:,} parameters")
print(f"Task: sort lists of {train_dataset.length} digits ({train_dataset.num_digits} possible values)")
print(f"Example: [2, 0, 1] → [0, 1, 2]\n")

# Setup (mirrors mingpt/trainer.py)
device = 'cuda' if torch.cuda.is_available() else 'cpu'
model = model.to(device)
optimizer = torch.optim.AdamW(model.parameters(), lr=5e-4, weight_decay=0.1)

# Training loop
train_loader = DataLoader(train_dataset, batch_size=64, num_workers=0,
                          sampler=torch.utils.data.RandomSampler(train_dataset, replacement=True, num_samples=int(1e10)))
data_iter = iter(train_loader)

max_iters = 1000
print(f"Training for {max_iters} iterations on {device}...\n")

model.train()
t0 = time.time()

for iter_num in range(max_iters):
    # ---- Step 1: Get batch ----
    batch = next(data_iter)
    x, y = [t.to(device) for t in batch]

    # ---- Step 2: Forward pass ----
    logits, loss = model(x, y)

    # ---- Step 3-4: Backward pass ----
    model.zero_grad(set_to_none=True)
    loss.backward()

    # ---- Step 5: Gradient clipping (prevent explosions) ----
    torch.nn.utils.clip_grad_norm_(model.parameters(), 1.0)

    # ---- Step 6: Optimizer step ----
    optimizer.step()

    if iter_num % 100 == 0:
        elapsed = time.time() - t0
        print(f"  iter {iter_num:4d} | loss {loss.item():.5f} | time {elapsed:.1f}s")

elapsed = time.time() - t0
print(f"\nTraining complete in {elapsed:.1f}s")

# ============================================================
# EVALUATION: Test if it actually learned to sort!
# ============================================================
print("\n" + "=" * 60)
print("Evaluation: Can it sort?")
print("=" * 60)

model.eval()
n = train_dataset.length

def eval_split(split, max_batches=50):
    dataset = train_dataset if split == 'train' else test_dataset
    loader = DataLoader(dataset, batch_size=100, num_workers=0)
    correct = 0
    total = 0
    for b, (x, y) in enumerate(loader):
        x = x.to(device)
        inp = x[:, :n]
        with torch.no_grad():
            cat = model.generate(inp, n)
        sol_pred = cat[:, n:]
        sol_true = torch.sort(inp)[0]
        correct += (sol_pred.cpu() == sol_true).all(1).sum().item()
        total += x.size(0)
        if b + 1 >= max_batches:
            break
    return correct, total

train_correct, train_total = eval_split('train')
test_correct, test_total = eval_split('test')

print(f"\n  Train: {train_correct}/{train_total} = {100*train_correct/train_total:.1f}% correct")
print(f"  Test:  {test_correct}/{test_total} = {100*test_correct/test_total:.1f}% correct")

# Show some examples
print("\n  Sample predictions:")
loader = DataLoader(test_dataset, batch_size=5, num_workers=0)
x, y = next(iter(loader))
inp = x[:, :n].to(device)
with torch.no_grad():
    cat = model.generate(inp, n)
for i in range(5):
    input_list = inp[i].tolist()
    pred_list = cat[i, n:].tolist()
    true_list = sorted(input_list)
    status = "✓" if pred_list == true_list else "✗"
    print(f"    {input_list} → {pred_list} (expected {true_list}) {status}")

print("""
=== What This Demonstrates ===

GPT learned to SORT numbers — a task it was never explicitly programmed for!
It learned the algorithm purely from examples:
  - See thousands of (unsorted → sorted) pairs
  - Learn the pattern through gradient descent
  - Generalize to new, unseen inputs

This is the power of the transformer architecture: given enough data
and parameters, it can learn surprisingly complex functions.

The same architecture, scaled up with more data and parameters,
learns to write code, answer questions, translate languages...
""")
