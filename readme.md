# Go CodeGraph CLI

A simple CLI tool that analyzes a Go project and generates a JSON “code graph” of packages, types, functions, and their relationships.

## ✨ Features

- Recursively parses Go files (skips `vendor/` and `_test.go`)
- Extracts structs, interfaces, functions, constants, variables, and dependencies
- Builds a graph of nodes (entities) and edges (relationships)
- Outputs a pretty-printed JSON file

## 🛠 Prerequisites

- Go 1.18 or newer installed and available in your `PATH`

## 📦 Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/srinidhi-metadome/go-codegraph-cli.git
   cd go-codegraph-cli


