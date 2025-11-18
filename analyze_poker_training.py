import pandas as pd
import matplotlib.pyplot as plt

# Load CSV
df = pd.read_csv("training_log.csv")

# Basic cleaning
df['episode'] = df['episode'].astype(int)

# --- 1. Epsilon Decay ---
plt.figure(figsize=(10,5))
plt.plot(df['episode'], df['epsilon'])
plt.title("Epsilon Decay Over Training")
plt.xlabel("Episode")
plt.ylabel("Epsilon")
plt.grid(True)
plt.show()

# --- 2. Win Rate (sliding window) ---
df['winrate_5k'] = df['wins'].rolling(window=5).mean()

plt.figure(figsize=(10,5))
plt.plot(df['episode'], df['winrate_5k'])
plt.title("Bot Win Rate (Averaged per 5k Episodes)")
plt.xlabel("Episode")
plt.ylabel("Wins per 1000")
plt.grid(True)
plt.show()

# --- 3. Bot vs Clone Chips ---
plt.figure(figsize=(10,5))
plt.plot(df['episode'], df['bot_chips'], label="Bot")
plt.plot(df['episode'], df['clone_chips'], label="Clone")
plt.title("Chip Counts: Bot vs Clone")
plt.xlabel("Episode")
plt.ylabel("Stack Size")
plt.legend()
plt.grid(True)
plt.show()

# --- 4. Buyins (Idiot Moments) ---
plt.figure(figsize=(10,5))
plt.plot(df['episode'], df['bot_buyins'], label="Bot Buyins")
plt.plot(df['episode'], df['clone_buyins'], label="Clone Buyins")
plt.title("Buy-in Count (How Often They Dusted Their Stack)")
plt.xlabel("Episode")
plt.ylabel("Buy-ins")
plt.legend()
plt.grid(True)
plt.show()

# --- 5. Depth of Hands ---
plt.figure(figsize=(10,5))
plt.plot(df['episode'], df['flops'], label="Flops")
plt.plot(df['episode'], df['turns'], label="Turns")
plt.plot(df['episode'], df['rivers'], label="Rivers")
plt.title("Hand Depth (Did the Game Reach Flop/Turn/River?)")
plt.xlabel("Episode")
plt.ylabel("Count")
plt.legend()
plt.grid(True)
plt.show()

