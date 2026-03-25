---
name: run-python
description: Run Python scripts or one-liners using uv. Use when python3 is not directly available in the environment.
user-invocable: false
allowed-tools: Bash(uv run python*)
---

This environment does not have `python3` on PATH. Use `uv run python` instead.

**One-liner:**
```bash
uv run python -c "print('hello')"
```

**Script file:**
```bash
uv run python script.py
```

**With stdin pipe:**
```bash
some-command | uv run python -c "import sys; [print(line.strip()) for line in sys.stdin]"
```
