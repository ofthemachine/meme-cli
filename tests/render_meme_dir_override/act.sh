#!/bin/sh
set -e

mkdir -p memedir/custom
cat > memedir/custom/config.yaml <<'EOF'
name: "Custom"
background: {color: "#222222", width: 100, height: 100}
text_boxes: [{name: a, x: 0, y: 0, width: 1, height: 1}]
EOF

# The bundled library has no "custom" template, so this must fail without
# --meme-dir, proving the override below actually takes effect.
echo "=== without --meme-dir ==="
meme-cli show custom >/dev/null 2>&1 && echo "unexpected success" || echo "failed as expected"

echo "=== with --meme-dir ==="
meme-cli --meme-dir memedir render custom hi -o out.png
test -s out.png && echo "file non-empty"
