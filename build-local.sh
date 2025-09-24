#!/usr/bin/env bash

# Build cs from local source and install to ~/.local/bin
echo "Building cs from local source..."
go build -o ~/.local/bin/cs .
echo "Done! Local cs installed to ~/.local/bin/cs"
cs version