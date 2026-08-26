#!/usr/bin/env python3
"""Local editor for marking quad write-area corners on templates.

Usage:
    python3 scripts/parallelogram_editor.py [--port 8765] [--open]

Serves a click-to-mark UI plus the templates/ tree. Existing text_boxes from
config.yaml are drawn when you pick a template.
"""

from __future__ import annotations

import argparse
import json
import math
import mimetypes
import sys
import threading
import urllib.parse
import webbrowser
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

try:
    import yaml
except ImportError:
    print("PyYAML required: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

ROOT = Path(__file__).resolve().parent.parent
TEMPLATES = ROOT / "templates"
HTML = Path(__file__).resolve().parent / "parallelogram_editor.html"


def list_templates() -> list[dict]:
    out = []
    if not TEMPLATES.is_dir():
        return out
    for d in sorted(TEMPLATES.iterdir()):
        if not d.is_dir():
            continue
        cfg = d / "config.yaml"
        bg = None
        for name in ("default.jpg", "default.jpeg", "default.png", "default.webp"):
            if (d / name).is_file():
                bg = name
                break
        if bg is None:
            continue
        out.append({"id": d.name, "image": bg, "has_config": cfg.is_file()})
    return out


def _as_point(p) -> dict | None:
    if isinstance(p, (list, tuple)) and len(p) == 2:
        return {"x": float(p[0]), "y": float(p[1])}
    if isinstance(p, dict) and "x" in p and "y" in p:
        return {"x": float(p["x"]), "y": float(p["y"])}
    return None


def _imply_br(tl: dict, tr: dict, bl: dict) -> dict:
    return {
        "x": tl["x"] + (tr["x"] - tl["x"]) + (bl["x"] - tl["x"]),
        "y": tl["y"] + (tr["y"] - tl["y"]) + (bl["y"] - tl["y"]),
    }


def _rotate_point(x: float, y: float, cx: float, cy: float, angle_deg: float) -> dict:
    # Positive angle = CCW (Pillow / memegen), y-down image coords.
    rad = math.radians(angle_deg)
    cos, sin = math.cos(rad), math.sin(rad)
    dx, dy = x - cx, y - cy
    return {"x": cx + dx * cos - dy * sin, "y": cy + dx * sin + dy * cos}


def _rect_corners(x: float, y: float, w: float, h: float, angle: float = 0.0) -> list[dict]:
    tl = {"x": x, "y": y}
    tr = {"x": x + w, "y": y}
    br = {"x": x + w, "y": y + h}
    bl = {"x": x, "y": y + h}
    if not angle:
        return [tl, tr, br, bl]
    cx, cy = x + w / 2, y + h / 2
    return [_rotate_point(p["x"], p["y"], cx, cy, angle) for p in (tl, tr, br, bl)]


def text_box_area(tb: dict) -> dict | None:
    """Normalize a text_box into {name, kind, corners[4], angle?} for the editor."""
    name = tb.get("name") or ""
    raw = tb.get("quad") or tb.get("parallelogram")
    if raw:
        pts = [_as_point(p) for p in raw]
        if any(p is None for p in pts):
            return None
        if len(pts) == 3:
            tl, tr, bl = pts
            corners = [tl, tr, _imply_br(tl, tr, bl), bl]
            return {"name": name, "kind": "quad", "corners": corners, "source": "3pt"}
        if len(pts) == 4:
            return {"name": name, "kind": "quad", "corners": pts, "source": "4pt"}
        return None

    w = float(tb.get("width") or 0)
    h = float(tb.get("height") or 0)
    if w <= 0 or h <= 0:
        return None
    x = float(tb.get("x") or 0)
    y = float(tb.get("y") or 0)
    angle = float(tb.get("angle") or 0)
    return {
        "name": name,
        "kind": "rect",
        "corners": _rect_corners(x, y, w, h, angle),
        "angle": angle,
        "source": "rect",
    }


def load_template(tmpl_id: str) -> dict | None:
    d = (TEMPLATES / tmpl_id).resolve()
    try:
        d.relative_to(TEMPLATES.resolve())
    except ValueError:
        return None
    if not d.is_dir():
        return None
    cfg_path = d / "config.yaml"
    if not cfg_path.is_file():
        return None
    raw = yaml.safe_load(cfg_path.read_text()) or {}
    bg = None
    for name in ("default.jpg", "default.jpeg", "default.png", "default.webp"):
        if (d / name).is_file():
            bg = name
            break
    areas = []
    for tb in raw.get("text_boxes") or []:
        if not isinstance(tb, dict):
            continue
        area = text_box_area(tb)
        if area:
            areas.append(area)
    return {
        "id": tmpl_id,
        "name": raw.get("name") or tmpl_id,
        "image": bg,
        "areas": areas,
    }


class Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt: str, *args) -> None:
        sys.stderr.write("%s - %s\n" % (self.address_string(), fmt % args))

    def _send(self, code: int, body: bytes, content_type: str) -> None:
        self.send_response(code)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self) -> None:
        parsed = urllib.parse.urlparse(self.path)
        path = urllib.parse.unquote(parsed.path)

        if path in ("/", "/index.html", "/editor"):
            if not HTML.is_file():
                self._send(500, b"missing parallelogram_editor.html", "text/plain")
                return
            self._send(200, HTML.read_bytes(), "text/html; charset=utf-8")
            return

        if path == "/api/templates":
            body = json.dumps(list_templates()).encode()
            self._send(200, body, "application/json")
            return

        if path.startswith("/api/templates/"):
            tmpl_id = path[len("/api/templates/") :].strip("/")
            if not tmpl_id or "/" in tmpl_id:
                self._send(400, b"bad id", "text/plain")
                return
            data = load_template(tmpl_id)
            if data is None:
                self._send(404, b"not found", "text/plain")
                return
            self._send(200, json.dumps(data).encode(), "application/json")
            return

        if path.startswith("/templates/"):
            rel = path[len("/templates/") :]
            candidate = (TEMPLATES / rel).resolve()
            try:
                candidate.relative_to(TEMPLATES.resolve())
            except ValueError:
                self._send(403, b"forbidden", "text/plain")
                return
            if not candidate.is_file():
                self._send(404, b"not found", "text/plain")
                return
            ctype = mimetypes.guess_type(str(candidate))[0] or "application/octet-stream"
            self._send(200, candidate.read_bytes(), ctype)
            return

        self._send(404, b"not found", "text/plain")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--port", type=int, default=8765)
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--open", action="store_true", help="open browser")
    args = parser.parse_args()

    if not HTML.is_file():
        print(f"missing {HTML}", file=sys.stderr)
        return 1

    server = ThreadingHTTPServer((args.host, args.port), Handler)
    url = f"http://{args.host}:{args.port}/"
    print(f"Quad editor: {url}")
    print(f"Templates root: {TEMPLATES}")
    print("Existing text_boxes are drawn on load. Ctrl+C to stop.")

    if args.open:
        threading.Timer(0.4, lambda: webbrowser.open(url)).start()

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nbye")
        server.shutdown()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
