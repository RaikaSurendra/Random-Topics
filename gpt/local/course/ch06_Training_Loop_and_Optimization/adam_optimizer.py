"""
Chapter 06: The Adam Optimizer

Adam is the optimizer used in both microGPT and minGPT.
It's smarter than plain gradient descent because it adapts
the learning rate for each parameter individually.

This demo compares plain SGD vs Adam on a simple problem.
"""

import math
import random

random.seed(42)

# ============================================================
# Problem: Minimize f(x, y) = (x - 3)^2 + 10*(y - 7)^2
# The minimum is at (3, 7). Can Adam find it?
# ============================================================

def f(x, y):
    """Function to minimize. Minimum at (3, 7)."""
    return (x - 3) ** 2 + 10 * (y - 7) ** 2

def grad_f(x, y):
    """Gradient of f."""
    return 2 * (x - 3), 20 * (y - 7)

# ============================================================
# Method 1: Plain Gradient Descent (SGD)
# ============================================================
print("=" * 60)
print("Method 1: Plain Gradient Descent (SGD)")
print("=" * 60)

x, y = 0.0, 0.0  # start far from the minimum
lr = 0.01

for step in range(50):
    gx, gy = grad_f(x, y)
    x -= lr * gx
    y -= lr * gy
    if step % 5 == 0:
        print(f"  Step {step:3d}: x={x:.4f}, y={y:.4f}, f={f(x, y):.4f}")

print(f"  Final: x={x:.4f} (target: 3.0), y={y:.4f} (target: 7.0)")

# ============================================================
# Method 2: Adam Optimizer
# ============================================================
print("\n" + "=" * 60)
print("Method 2: Adam Optimizer")
print("=" * 60)

x, y = 0.0, 0.0  # same starting point
lr = 0.1
beta1, beta2, eps = 0.9, 0.99, 1e-8

# Adam's extra state: momentum and velocity for each parameter
mx, my = 0.0, 0.0  # first moment (momentum)
vx, vy = 0.0, 0.0  # second moment (velocity)

for step in range(50):
    gx, gy = grad_f(x, y)

    # Update momentum (exponential moving average of gradients)
    mx = beta1 * mx + (1 - beta1) * gx
    my = beta1 * my + (1 - beta1) * gy

    # Update velocity (exponential moving average of squared gradients)
    vx = beta2 * vx + (1 - beta2) * gx ** 2
    vy = beta2 * vy + (1 - beta2) * gy ** 2

    # Bias correction (important for early steps when m and v are near 0)
    mx_hat = mx / (1 - beta1 ** (step + 1))
    my_hat = my / (1 - beta1 ** (step + 1))
    vx_hat = vx / (1 - beta2 ** (step + 1))
    vy_hat = vy / (1 - beta2 ** (step + 1))

    # Update parameters
    x -= lr * mx_hat / (vx_hat ** 0.5 + eps)
    y -= lr * my_hat / (vy_hat ** 0.5 + eps)

    if step % 5 == 0:
        print(f"  Step {step:3d}: x={x:.4f}, y={y:.4f}, f={f(x, y):.4f}")

print(f"  Final: x={x:.4f} (target: 3.0), y={y:.4f} (target: 7.0)")

# ============================================================
# Explain what Adam does
# ============================================================
print(f"""
=== Why Adam is Better ===

Plain SGD: every parameter uses the SAME learning rate.
  - The y direction has steep gradients (10x) → it overshoots or oscillates
  - The x direction has gentle gradients → it moves too slowly
  - You can't make both happy with one learning rate

Adam: ADAPTS the learning rate per parameter.
  - Momentum (m): smooths out gradient noise, like a rolling ball
  - Velocity (v): tracks gradient magnitude per parameter
  - Parameters with large gradients get SMALLER steps
  - Parameters with small gradients get LARGER steps

=== Adam in microGPT (line 176-182) ===

  for i, p in enumerate(params):
      m[i] = beta1 * m[i] + (1 - beta1) * p.grad         # momentum
      v[i] = beta2 * v[i] + (1 - beta2) * p.grad ** 2    # velocity
      m_hat = m[i] / (1 - beta1 ** (step + 1))            # bias correction
      v_hat = v[i] / (1 - beta2 ** (step + 1))            # bias correction
      p.data -= lr * m_hat / (v_hat ** 0.5 + eps)         # update!
      p.grad = 0                                            # reset for next step

Same algorithm, just applied to thousands of parameters at once.

=== Learning Rate Decay ===

microGPT also decays the learning rate linearly:
  lr_t = lr * (1 - step / total_steps)

This means: take big steps early (explore), small steps late (fine-tune).
""")
