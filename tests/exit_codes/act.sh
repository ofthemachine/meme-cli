#!/bin/sh
# No `set -e`: every command below is expected to fail, and we're asserting
# on the exit code, not just non-zero-ness.

meme-cli show not-a-real-template >/dev/null 2>&1
echo "unknown template exit: $?"

meme-cli --meme-dir /does/not/exist list >/dev/null 2>&1
echo "bad meme-dir exit: $?"

meme-cli show >/dev/null 2>&1
echo "bad args exit: $?"
