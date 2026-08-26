#!/bin/sh
set -e

meme-cli render quote-card "integration test" -o out.png
test -s out.png && echo "file non-empty"
