#!/usr/bin/env python3
"""Patch meme-cli template text_boxes from memegen.link config.yml geometry.

Memegen uses anchor_x/anchor_y (top-left) + scale_x/scale_y (fraction of image).
This script converts those to meme-cli's x/y/width/height boxes and preserves
color, uppercase, and stroke hints where possible.

Usage:
    python3 scripts/sync_memegen_layouts.py [--render-examples]
"""

from __future__ import annotations

import argparse
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path

try:
    import yaml
except ImportError:
    print("PyYAML required: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

ROOT = Path(__file__).resolve().parent.parent
TEMPLATES = ROOT / "templates"
EXAMPLES = ROOT / "examples"
MEMEGEN_CFG = (
    "https://raw.githubusercontent.com/jacebrowning/memegen/main/templates/{id}/config.yml"
)

# memegen-rs applies a horizontal inset inside each text band.
SAFE_MARGIN = 0.05

# Original seed + wikimedia — leave layout alone.
SKIP = {
    "quote-card", "the-scream", "mona-lisa", "great-wave", "creation-of-adam",
    "vitruvian-man", "earthrise", "moon-landing", "pearl-earring",
    "napoleon-alps", "liberty-leading", "starry-night", "whistlers-mother",
    "guernica", "pillars-of-creation", "pale-blue-dot", "black-hole",
    "klimt-kiss", "sunday-afternoon", "nighthawks", "rosie-riveter",
    "uncle-sam", "keep-calm", "alien-autopsy-still", "persistence-of-memory",
    # 3-panel vertical comic — memegen's full-width bands cover faces; tune manually.
    "gandalf",
    "bilbo",
    # cmm: y nudged vs memegen (our rotate paste is center-preserving; Pillow expand+paste sits lower).
    "cmm",
    # crowd: animated overlay in memegen; static stack is hand-tuned.
    "crowd",
}

COLOR_MAP = {
    "khaki": "#C3B091",
}


def fetch_memegen_config(tid: str) -> dict | None:
    url = MEMEGEN_CFG.format(id=tid)
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "meme-cli-sync/1.0"})
        with urllib.request.urlopen(req, timeout=30) as resp:
            return yaml.safe_load(resp.read())
    except (urllib.error.URLError, TimeoutError, yaml.YAMLError):
        return None


def convert_text_entry(entry: dict, index: int) -> dict:
    ax = float(entry.get("anchor_x", 0))
    ay = float(entry.get("anchor_y", 0))
    sx = float(entry.get("scale_x", 1.0))
    sy = float(entry.get("scale_y", 0.2))

    style = (entry.get("style") or "upper").lower()
    uppercase = style == "upper"

    color = entry.get("color") or "white"
    color = COLOR_MAP.get(color, color)

    font = (entry.get("font") or "thick").lower()
    stroke_width = 3
    if font == "thin" or color in ("black", "#000000", "#000"):
        stroke_width = 0

    names = ["top", "bottom", "middle", "line4", "line5", "line6", "line7", "line8"]
    name = names[index] if index < len(names) else f"line{index + 1}"

    box: dict = {
        "name": name,
        "x": round(ax + SAFE_MARGIN * sx, 4),
        "y": round(ay, 4),
        "width": round(max(sx * (1 - 2 * SAFE_MARGIN), 0.01), 4),
        "height": round(max(sy, 0.01), 4),
        "align": entry.get("align") or "center",
        "valign": "middle",
    }
    if uppercase:
        box["uppercase"] = True
    if color and color.lower() not in ("white",):
        box["color"] = color
    if stroke_width == 0:
        box["stroke_width"] = 0
    angle = entry.get("angle")
    if angle:
        box["angle"] = round(float(angle), 2)
    return box


def sync_template(tid: str) -> bool:
    mg = fetch_memegen_config(tid)
    if not mg or not mg.get("text"):
        return False

    cfg_path = TEMPLATES / tid / "config.yaml"
    if not cfg_path.exists():
        return False

    with open(cfg_path) as f:
        cfg = yaml.safe_load(f)

    cfg["text_boxes"] = [convert_text_entry(e, i) for i, e in enumerate(mg["text"])]

    with open(cfg_path, "w") as f:
        yaml.dump(cfg, f, default_flow_style=False, sort_keys=False, allow_unicode=True)
    return True


def render_example(tid: str, binary: Path) -> bool:
    cfg_path = TEMPLATES / tid / "config.yaml"
    with open(cfg_path) as f:
        cfg = yaml.safe_load(f)
    texts = cfg.get("example_text") or []
    if not texts or all(not str(t).strip() for t in texts):
        return False
    out = EXAMPLES / tid / "eg01.png"
    out.parent.mkdir(parents=True, exist_ok=True)
    try:
        subprocess.run(
            [str(binary), "--meme-dir", str(TEMPLATES), "render", tid,
             *[str(t) for t in texts], "-o", str(out)],
            cwd=ROOT,
            check=True,
            capture_output=True,
            text=True,
        )
        return True
    except subprocess.CalledProcessError as e:
        print(f"  render FAIL {tid}: {e.stderr or e}", file=sys.stderr)
        return False


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--render-examples", action="store_true")
    parser.add_argument("--id", help="sync only this template id")
    args = parser.parse_args()

    ids = [args.id] if args.id else sorted(p.name for p in TEMPLATES.iterdir() if p.is_dir())
    ok, miss = 0, 0
    for tid in ids:
        if tid in SKIP:
            continue
        if sync_template(tid):
            print(f"  synced {tid}")
            ok += 1
        else:
            miss += 1

    print(f"Synced {ok} templates ({miss} skipped or missing memegen config)")

    if args.render_examples:
        binary = ROOT / "meme-cli"
        if not binary.exists():
            subprocess.run(["make", "build"], cwd=ROOT, check=True)
        rendered = 0
        for tid in ids:
            if render_example(tid, binary):
                rendered += 1
        print(f"Re-rendered {rendered} examples")


if __name__ == "__main__":
    main()
