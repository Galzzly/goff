# goff

<p align="center">
<img src=./template/readme/logo.png>
</p>

Go template for a fast and efficient FlipFlip Codes

## Setup

Run the interactive setup to configure the repo and install the tool:

```sh
./setup.sh
```

## CLI

Build the helper tool manually:

```sh
go build -o goff ./templates/cmd/goff
```

Usage:

```sh
# Store session token (PHPSESSID)
./goff s "<token>"

# Prepare a puzzle (defaults to current year)
./goff p 1

# Prepare a puzzle for a specific year
./goff p -y 2025 1
./goff p y 2025 1

# Get puzzle text for the current directory
./goff g
./goff g 2
./goff g 3
./goff g 4

# Run with input or test (current directory)
./goff r i
./goff r t

# Benchmark (current directory; always uses input)
./goff b

# Download input for a puzzle (requires stored PHPSESSID)
./goff d 1

# Update pointer + benchmark summary in README (requires stored PHPSESSID)
./goff summary
```

<!-- GOFF:POINTERS:START -->
# Flip Flop

## Year : 2025

### Pointers

Pointers (2025): 0/21

### Benchmarks

No benchmarks yet.
<!-- GOFF:POINTERS:END -->

To keep the pointer/benchmark summary updated on push, add a repository secret named `FLIPFLOP_PHPSESSID`.
