# Go Function Extraction Tool

This tool allows you to extract a specified function along with its top-level dependencies from a Go project into a single output file. It parses the Go files in a given directory, identifies the requested function and any functions, types, variables, or constants that it depends on, and then writes them to a new file for easier inspection.

## Example: Extracting the `DoStuff` Function

### Prerequisites

- Go installed (version 1.18+ recommended)
- A directory containing Go source files and a target function you want to extract.

### Example Files

For demonstration purposes, suppose you have the following files in the `example` directory:

- `main.go`
- `helpers.go`
- `helpers2.go`
- `types.go`

### Usage

After go run main.go

When prompted:

Directory to analyze: Type example for the exaple directory (/example).
Name of the function to extract: Enter DoStuff.
Output file: Press Enter for the default output.go.
The tool will parse all .go files in example and find DoStuff along with all its dependencies (helperFunc, MyType, MyVar, and MyConst) and write them to output.go.

Checking the Output
After running the tool, you'll have an output.go file containing:

The package declaration (e.g., package example)
The DoStuff function
All top-level dependencies (helperFunc, MyType, MyVar, MyConst)
You can then use output.go as a self-contained file providing all the code needed for context analysis or feeding into another tool (like a Large Language Model) for code review, analysis, or summarization.

@lube ➜ /workspaces/fextract (main) $ go run main.go
Directory to analyze [default: .]: example
Name of the function to extract: DoStuff
Output file [default: output.go]: 
Extraction completed. Code written to 'output.go'.