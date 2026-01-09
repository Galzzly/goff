# goff

![goff logo](./template/readme/logo.png)

A helper CLI tool for managing FlipFlip Codes puzzles efficiently.

## Getting Started

### Initial Setup

1. Fork this repository
2. Clone your fork locally
3. Run the interactive setup script:

```sh
./setup.sh
```

This will:

- Prompt you to select a year to work on
- Ask for your PHPSESSID session token (optional, needed for downloading inputs)
- Build the `goff` binary
- Optionally install GitHub workflow files
- Clean up the build scaffolding
- Remove itself

After setup completes, you'll have a `goff` binary ready to use!

## Usage

The `goff` tool provides commands for managing your puzzle work:

### Session Management

Store your PHPSESSID session token (needed for downloading inputs):

```sh
goff session "<token>"
goff s "<token>"  # Short alias
```

### Preparing Puzzles

Create a new puzzle directory with templates. By default, puzzles are prepared for the year you're currently working on (stored in your config):

```sh
# Prepare puzzle 1 for your active working year
goff prepare 1
goff p 1  # Short alias

# Prepare puzzle for a different year
goff prepare -y 2025 1
goff p y 2025 1

# Note: You can change your active working year with `goff setup <year>`
```

### Viewing Puzzle Text

Get the puzzle description:

```sh
# From within a puzzle directory (auto-detects year/day)
goff get
goff g

# Get a specific part (1 or 2)
goff get 2
goff g 2
```

### Running Solutions

Execute your puzzle solution:

```sh
# From within a puzzle directory
goff run i   # Run with real input
goff run t   # Run with test input
goff r i
goff r t
```

### Benchmarking

Benchmark your solution:

```sh
# From within a puzzle directory (always uses real input)
goff bench
goff b
```

### Downloading Inputs

Download puzzle inputs (requires PHPSESSID):

```sh
goff download 1
goff d 1
```

### Updating Summaries

Update pointer counts and benchmarks in your README (requires PHPSESSID):

```sh
goff summary
```

### Setting Up New Years

After the initial setup, use this to set up a new year:

```sh
goff setup      # Sets up the current year
goff setup 2027 # Sets up a specific year
goff init       # Alias for setup
```

## Templates

The puzzle templates are stored in `template/go/puzzle/` and can be customized to match your preferred solution structure. Templates are applied when you run `goff prepare`.

## Project Structure

After setup, your repository will contain:

- `goff` - The compiled binary
- `template/go/puzzle/` - Customizable puzzle templates
- `template/README.year.tmpl` - Template for year README files
- Year directories (e.g., `2025/`, `2026/`) - Your puzzle solutions
