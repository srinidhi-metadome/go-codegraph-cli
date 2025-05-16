# Go CodeGraph CLI

A simple CLI tool that analyzes a Go project and generates a JSON “code graph” of packages, types, functions, and their relationships.

## Features

- Recursively parses Go files (skips `vendor/` and `_test.go`)  
- Extracts structs, interfaces, functions, constants, variables, and dependencies  
- Builds a graph of nodes (entities) and edges (relationships)  
- Outputs a pretty-printed JSON file

## Prerequisites

- Go 1.18 or newer installed and on your `PATH`

## Installation

1. Clone this repo:
   ```bash
   git clone https://github.com/srinidhi-metadome/go-codegraph-cli.git
   cd go-codegraph-cli


## Tidy modules and build the binary:


go mod tidy
go build -o codegraph .

## Usage

# Default: analyze current directory, project name "MyProject", output to output.json
./codegraph

# Custom path, project name, and output file
./codegraph \
  --path /path/to/your/project \
  --name MyProjectName \
  --output my-project-graph.json

# View help
./codegraph --help

## Example

./codegraph \
  --path ../example-go-app \
  --name ExampleApp \
  --output example-graph.json

