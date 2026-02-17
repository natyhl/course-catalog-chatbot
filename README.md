# Course Catalog Chatbot (Go + Embeddings)

A Go-based course search + chat assistant that helps users find university courses using semantic search (embeddings) and a lightweight local database.

## Features
- Semantic course search using embeddings (vector similarity)
- Natural-language Q&A over course descriptions and metadata
- Simple CLI workflow (easy to run locally)
- Local DB storage (excluded from git via `.gitignore`)

## Tech Stack
- Go
- OpenAI API (embeddings / chat)
- Local database (SQLite or similar; not committed)

## Project Structure
> Adjust these filenames if yours differ.

- `main.go` — entry point
- `db.go` — DB + client initialization
- `embeddings.go` — embedding creation / storage / similarity search
- `chat.go` — chat request + tool/function logic (if used)
- `courses.csv` — input dataset (if allowed to publish)
- `*_test.go` — unit tests

## Setup

### 1) Clone
```bash
git clone git@github.com:natyhl/<REPO_NAME>.git
cd <REPO_NAME>
