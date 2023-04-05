# Git AI Commit

A git plugin to automatically generate commit messages using GPT.

## Installation

```bash
go install github.com/graytonio/git-ai-commit@latest
```

## Usage

```bash
# Ensure OPENAI_TOKEN is set in env
export OPENAI_TOKEN="my-openai-token"

Usage of git-ai-commit:
  -dry
        process diff without making commit
  -prefix string
        prefix for commit message
  -suffix string
        suffix for commit message
  -verbose
        verbose logging
```
