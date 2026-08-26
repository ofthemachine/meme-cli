#!/usr/bin/env python3
"""Build examples/gallery.html — a single page to review all example memes.

Usage:
    python3 scripts/generate_gallery.py [--open]

Scans examples/<template-id>/eg01.png, pulls names from templates/*/config.yaml,
and writes a self-contained gallery (relative image paths; open via file:// or
any static server rooted at examples/).
"""

from __future__ import annotations

import argparse
import html
import subprocess
import sys
import webbrowser
from datetime import datetime, timezone
from pathlib import Path

try:
    import yaml
except ImportError:
    print("PyYAML required: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

ROOT = Path(__file__).resolve().parent.parent
TEMPLATES = ROOT / "templates"
EXAMPLES = ROOT / "examples"
OUT = EXAMPLES / "gallery.html"


def load_meta(tid: str) -> tuple[str, list[str]]:
    cfg_path = TEMPLATES / tid / "config.yaml"
    if not cfg_path.exists():
        return tid, []
    with open(cfg_path) as f:
        cfg = yaml.safe_load(f) or {}
    return cfg.get("name") or tid, cfg.get("example_text") or []


def build_html(cards: list[dict]) -> str:
    generated = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    rows = []
    for c in cards:
        lines = "".join(
            f'<li>{html.escape(line)}</li>' for line in c["lines"] if line.strip()
        )
        if not lines:
            lines = "<li><em>(no caption)</em></li>"
        rows.append(
            f"""<article class="card" id="{html.escape(c["id"])}">
  <a href="#{html.escape(c["id"])}" class="thumb-link">
    <img src="{html.escape(c["src"])}" alt="{html.escape(c["name"])}" loading="lazy">
  </a>
  <div class="meta">
    <h2><code>{html.escape(c["id"])}</code></h2>
    <p class="name">{html.escape(c["name"])}</p>
    <ul class="caption">{lines}</ul>
  </div>
</article>"""
        )

    body = "\n".join(rows)
    return f"""<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>meme-cli example gallery ({len(cards)} templates)</title>
  <style>
    :root {{
      color-scheme: dark light;
      --bg: #0f0f0f;
      --card: #1a1a1a;
      --text: #e8e8e8;
      --muted: #888;
      --accent: #6cf;
    }}
    @media (prefers-color-scheme: light) {{
      :root {{ --bg: #f4f4f4; --card: #fff; --text: #111; --muted: #666; --accent: #06c; }}
    }}
    * {{ box-sizing: border-box; }}
    body {{
      margin: 0;
      font-family: system-ui, -apple-system, sans-serif;
      background: var(--bg);
      color: var(--text);
      line-height: 1.4;
    }}
    header {{
      position: sticky;
      top: 0;
      z-index: 1;
      padding: 1rem 1.5rem;
      background: color-mix(in srgb, var(--bg) 92%, transparent);
      backdrop-filter: blur(8px);
      border-bottom: 1px solid color-mix(in srgb, var(--text) 12%, transparent);
    }}
    header h1 {{ margin: 0 0 0.25rem; font-size: 1.25rem; }}
    header p {{ margin: 0; color: var(--muted); font-size: 0.875rem; }}
    #filter {{
      margin-top: 0.75rem;
      width: min(28rem, 100%);
      padding: 0.5rem 0.75rem;
      border: 1px solid color-mix(in srgb, var(--text) 20%, transparent);
      border-radius: 6px;
      background: var(--card);
      color: var(--text);
      font-size: 1rem;
    }}
    main {{
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 1.25rem;
      padding: 1.5rem;
      max-width: 1800px;
      margin: 0 auto;
    }}
    .card {{
      background: var(--card);
      border-radius: 10px;
      overflow: hidden;
      border: 1px solid color-mix(in srgb, var(--text) 10%, transparent);
      display: flex;
      flex-direction: column;
    }}
    .card.hidden {{ display: none; }}
    .thumb-link {{ display: block; background: #000; }}
    .card img {{
      width: 100%;
      height: auto;
      display: block;
      vertical-align: middle;
    }}
    .meta {{ padding: 0.75rem 1rem 1rem; }}
    .meta h2 {{ margin: 0 0 0.25rem; font-size: 0.95rem; }}
    .meta code {{
      color: var(--accent);
      font-size: 0.9rem;
      word-break: break-all;
    }}
    .name {{ margin: 0 0 0.5rem; font-weight: 600; }}
    .caption {{
      margin: 0;
      padding-left: 1.1rem;
      font-size: 0.8rem;
      color: var(--muted);
    }}
    .caption li {{ margin-bottom: 0.15rem; }}
    footer {{
      text-align: center;
      padding: 2rem 1rem;
      color: var(--muted);
      font-size: 0.8rem;
    }}
  </style>
</head>
<body>
  <header>
    <h1>meme-cli example gallery</h1>
    <p>{len(cards)} templates · generated {generated}</p>
    <input id="filter" type="search" placeholder="Filter by id or name…" autocomplete="off">
  </header>
  <main id="grid">
{body}
  </main>
  <footer>
    Regenerate: <code>python3 scripts/generate_gallery.py</code> or <code>make gallery</code>
  </footer>
  <script>
    const filter = document.getElementById('filter');
    const cards = [...document.querySelectorAll('.card')];
    filter.addEventListener('input', () => {{
      const q = filter.value.trim().toLowerCase();
      for (const card of cards) {{
        const text = card.textContent.toLowerCase();
        card.classList.toggle('hidden', q && !text.includes(q));
      }}
    }});
  </script>
</body>
</html>
"""


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--open", action="store_true", help="open gallery in browser")
    args = parser.parse_args()

    cards: list[dict] = []
    for img in sorted(EXAMPLES.glob("*/eg01.png")):
        tid = img.parent.name
        name, lines = load_meta(tid)
        cards.append({
            "id": tid,
            "name": name,
            "lines": lines,
            "src": f"{tid}/eg01.png",
        })

    if not cards:
        print("No examples found under examples/*/eg01.png", file=sys.stderr)
        print("Run: python3 scripts/bootstrap_templates.py --examples-only", file=sys.stderr)
        sys.exit(1)

    EXAMPLES.mkdir(parents=True, exist_ok=True)
    OUT.write_text(build_html(cards), encoding="utf-8")
    print(f"Wrote {OUT} ({len(cards)} memes)")

    if args.open:
        webbrowser.open(OUT.as_uri())


if __name__ == "__main__":
    main()
