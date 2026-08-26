#!/usr/bin/env python3
"""Bootstrap meme-cli templates from memegen.link and render one example meme per template.

Usage:
    python3 scripts/bootstrap_templates.py [--skip-download] [--examples-only]

Downloads blank template backgrounds, writes config.yaml files, and renders
a single custom example into examples/<template-id>/eg01.png (gitignored).
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import time
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
MEMEGEN_API = "https://api.memegen.link/templates/"
CUSTOM = Path(__file__).parent / "custom_examples.yaml"

# Existing seed templates — don't overwrite.
KEEP = {
    "quote-card", "the-scream", "mona-lisa", "great-wave", "creation-of-adam",
    "vitruvian-man", "earthrise", "moon-landing", "pearl-earring",
    "napoleon-alps", "liberty-leading",
}

# Skip animated / non-jpeg backgrounds Go can't decode without extra deps.
SKIP_IDS = set()

LICENSE = "Imported from jacebrowning/memegen (MIT); see NOTICE.md for template provenance."


def fetch_json(url: str) -> list | dict:
    req = urllib.request.Request(url, headers={"User-Agent": "meme-cli-bootstrap/1.0"})
    with urllib.request.urlopen(req, timeout=60) as resp:
        return json.loads(resp.read())


def download(url: str, dest: Path, retries: int = 3) -> bool:
    dest.parent.mkdir(parents=True, exist_ok=True)
    for attempt in range(retries):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "meme-cli-bootstrap/1.0"})
            with urllib.request.urlopen(req, timeout=120) as resp:
                dest.write_bytes(resp.read())
            return True
        except (urllib.error.URLError, TimeoutError) as e:
            if attempt == retries - 1:
                print(f"  FAIL download {url}: {e}")
                # Don't leave a dangling empty dir behind — it breaks the
                # //go:embed * in templates/embed.go on the next build.
                try:
                    dest.parent.rmdir()
                except OSError:
                    pass
                return False
            time.sleep(1 + attempt)
    return False


def text_boxes(lines: int) -> list[dict]:
    """Standard fractional text boxes by line count."""
    base = {"align": "center", "uppercase": True}
    if lines <= 1:
        return [{
            **base, "name": "text", "x": 0.05, "y": 0.32, "width": 0.9,
            "height": 0.36, "valign": "middle",
        }]
    if lines == 2:
        return [
            {**base, "name": "top", "x": 0.02, "y": 0.0, "width": 0.96,
             "height": 0.2, "valign": "top"},
            {**base, "name": "bottom", "x": 0.02, "y": 0.82, "width": 0.96,
             "height": 0.16, "valign": "bottom"},
        ]
    if lines == 3:
        return [
            {**base, "name": "top", "x": 0.02, "y": 0.0, "width": 0.96,
             "height": 0.14, "valign": "top"},
            {**base, "name": "middle", "x": 0.02, "y": 0.42, "width": 0.96,
             "height": 0.14, "valign": "middle"},
            {**base, "name": "bottom", "x": 0.02, "y": 0.86, "width": 0.96,
             "height": 0.12, "valign": "bottom"},
        ]
    # 4+ lines: stack evenly in upper 90%
    boxes = []
    n = min(lines, 8)
    h = 0.88 / n
    for i in range(n):
        boxes.append({
            **base,
            "name": f"line{i+1}",
            "x": 0.02,
            "y": 0.02 + i * h,
            "width": 0.96,
            "height": h * 0.85,
            "valign": "middle",
            "max_font_size": 36,
        })
    return boxes


def keywords_from(meta: dict) -> list[str]:
    kw = meta.get("keywords") or []
    kw = [k for k in kw if k]
    name = meta.get("name") or ""
    if name:
        kw.append(name.lower())
    # dedupe preserve order
    seen = set()
    out = []
    for k in kw:
        kl = k.lower().strip()
        if kl and kl not in seen:
            seen.add(kl)
            out.append(kl)
    return out[:12]


def fallback_examples(meta: dict) -> list[str]:
    lines = meta.get("lines") or 2
    name = meta.get("name") or meta["id"]
    if lines == 1:
        return [f"When {name} hits different at 3am"]
    if lines == 2:
        return ["Me explaining UFO physics to Foxhole squad", "Them nodding politely"]
    texts = [f"Foxhole logi line {i+1}" for i in range(min(lines, 4))]
    if lines >= 3:
        texts[-1] = "Still no disclosure"
    return texts


def import_memegen(skip_download: bool) -> tuple[int, int]:
    print("Fetching memegen template catalog...")
    catalog = fetch_json(MEMEGEN_API)
    by_id: dict[str, dict] = {}
    for t in catalog:
        tid = t["id"]
        if tid in by_id:
            continue
        blank = t.get("blank") or ""
        if blank.endswith(".webp") or blank.endswith(".gif"):
            SKIP_IDS.add(tid)
            continue
        by_id[tid] = t

    print(f"  {len(by_id)} static templates in catalog")
    ok, skip = 0, 0

    for tid, meta in sorted(by_id.items()):
        if tid in KEEP or tid in SKIP_IDS:
            skip += 1
            continue
        dest_dir = TEMPLATES / tid
        cfg_path = dest_dir / "config.yaml"
        img_path = dest_dir / "default.jpg"

        if not skip_download:
            if not img_path.exists():
                blank = meta["blank"]
                print(f"  download {tid}...")
                if not download(blank, img_path):
                    continue

        if not img_path.exists():
            print(f"  skip {tid}: no image")
            continue

        lines = meta.get("lines") or 2
        ex = meta.get("example") or {}
        example_text = ex.get("text") or fallback_examples(meta)

        cfg = {
            "name": meta.get("name") or tid,
            "source": meta.get("source") or f"https://api.memegen.link/templates/{tid}",
            "license": LICENSE,
            "keywords": keywords_from(meta),
            "background": {"image": "default.jpg"},
            "text_boxes": text_boxes(lines),
            "example_text": example_text[:lines] if len(example_text) > lines else example_text,
        }
        # pad example text if short
        while len(cfg["example_text"]) < lines:
            cfg["example_text"].append("")

        dest_dir.mkdir(parents=True, exist_ok=True)
        with open(cfg_path, "w") as f:
            yaml.dump(cfg, f, default_flow_style=False, sort_keys=False, allow_unicode=True)
        ok += 1

    return ok, skip


# Extra public-domain Wikimedia art / NASA templates beyond memegen.
WIKIMEDIA = [
    {
        "id": "starry-night",
        "name": "The Starry Night",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/e/ea/Van_Gogh_-_Starry_Night_-_Google_Art_Project.jpg/1280px-Van_Gogh_-_Starry_Night_-_Google_Art_Project.jpg",
        "source": "https://commons.wikimedia.org/wiki/File:Van_Gogh_-_Starry_Night_-_Google_Art_Project.jpg",
        "license": "Public domain (Vincent van Gogh, 1889)",
        "keywords": ["starry night", "van gogh", "ufo", "sky"],
    },
    {
        "id": "american-gothic",
        "name": "American Gothic",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/c/c7/Grant_Wood_-_American_Gothic_-_Google_Art_Project.jpg/800px-Grant_Wood_-_American_Gothic_-_Google_Art_Project.jpg",
        "source": "https://commons.wikimedia.org/wiki/File:Grant_Wood_-_American_Gothic_-_Google_Art_Project.jpg",
        "license": "Public domain (Grant Wood, 1930 — US work)",
        "keywords": ["american gothic", "pitchfork", "rural"],
    },
    {
        "id": "persistence-of-memory",
        "name": "The Persistence of Memory",
        "url": "https://upload.wikimedia.org/wikipedia/en/d/dd/The_Persistence_of_Memory.jpg",
        "source": "https://en.wikipedia.org/wiki/File:The_Persistence_of_Memory.jpg",
        "license": "Fair use / public domain status varies; Dali 1931",
        "keywords": ["dali", "clocks", "surreal"],
    },
    {
        "id": "whistlers-mother",
        "name": "Whistler's Mother",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/1/1b/Whistlers_Mother_high_res.jpg/800px-Whistlers_Mother_high_res.jpg",
        "source": "https://commons.wikimedia.org/wiki/File:Whistlers_Mother_high_res.jpg",
        "license": "Public domain (James McNeill Whistler, 1871)",
        "keywords": ["whistler", "mother", "waiting"],
    },
    {
        "id": "guernica",
        "name": "Guernica",
        "url": "https://upload.wikimedia.org/wikipedia/en/7/74/PicassoGuernica.jpg",
        "source": "https://en.wikipedia.org/wiki/File:PicassoGuernica.jpg",
        "license": "Public domain (Pablo Picasso, 1937)",
        "keywords": ["guernica", "picasso", "war", "chaos"],
    },
    {
        "id": "pillars-of-creation",
        "name": "Pillars of Creation",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/6/68/Pillars_of_creation_2014_HST_WFC3-UVIS_full-res_denoised.jpg/800px-Pillars_of_creation_2014_HST_WFC3-UVIS_full-res_denoised.jpg",
        "source": "https://commons.wikimedia.org/wiki/File:Pillars_of_creation_2014_HST_WFC3-UVIS_full-res_denoised.jpg",
        "license": "Public domain (NASA/ESA/Hubble)",
        "keywords": ["hubble", "space", "nasa", "nebula"],
    },
    {
        "id": "pale-blue-dot",
        "name": "Pale Blue Dot",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/7/73/Pale_Blue_Dot.png/800px-Pale_Blue_Dot.png",
        "source": "https://commons.wikimedia.org/wiki/File:Pale_Blue_Dot.png",
        "license": "Public domain (NASA/Voyager 1)",
        "keywords": ["pale blue dot", "sagan", "earth", "space"],
    },
    {
        "id": "black-hole",
        "name": "Black Hole (M87)",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/5/5f/Event_Horizon_Telescope_Collaboration_%28first_image_of_black_hole%29.jpg/800px-Event_Horizon_Telescope_Collaboration_%28first_image_of_black_hole%29.jpg",
        "source": "https://commons.wikimedia.org/wiki/File:Event_Horizon_Telescope_Collaboration_(first_image_of_black_hole).jpg",
        "license": "CC BY 4.0 (Event Horizon Telescope Collaboration)",
        "keywords": ["black hole", "m87", "space", "science"],
    },
    {
        "id": "klimt-kiss",
        "name": "The Kiss",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/4/40/The_Kiss_-_Gustav_Klimt_-_Google_Cultural_Institute.jpg/800px-The_Kiss_-_Gustav_Klimt_-_Google_Cultural_Institute.jpg",
        "source": "https://commons.wikimedia.org/wiki/File:The_Kiss_-_Gustav_Klimt_-_Google_Cultural_Institute.jpg",
        "license": "Public domain (Gustav Klimt, 1908)",
        "keywords": ["klimt", "kiss", "gold"],
    },
    {
        "id": "sunday-afternoon",
        "name": "A Sunday Afternoon on the Island of La Grande Jatte",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/7/7d/A_Sunday_on_La_Grande_Jatte%2C_Georges_Seurat%2C_1884.jpg/1280px-A_Sunday_on_La_Grande_Jatte%2C_Georges_Seurat%2C_1884.jpg",
        "source": "https://commons.wikimedia.org/wiki/File:A_Sunday_on_La_Grande_Jatte,_Georges_Seurat,_1884.jpg",
        "license": "Public domain (Georges Seurat, 1884)",
        "keywords": ["seurat", "park", "Sunday"],
    },
    {
        "id": "nighthawks",
        "name": "Nighthawks",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/a/a8/Nighthawks_by_Edward_Hopper_1942.jpg/1280px-Nighthawks_by_Edward_Hopper_1942.jpg",
        "source": "https://commons.wikimedia.org/wiki/File:Nighthawks_by_Edward_Hopper_1942.jpg",
        "license": "Public domain (Edward Hopper, 1942 — US work)",
        "keywords": ["hopper", "diner", "late night", "lonely"],
    },
    {
        "id": "rosie-riveter",
        "name": "We Can Do It!",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/1/1b/We_Can_Do_It%21.jpg/800px-We_Can_Do_It%21.jpg",
        "source": "https://commons.wikimedia.org/wiki/File:We_Can_Do_It!.jpg",
        "license": "Public domain (US government poster, J. Howard Miller, 1943)",
        "keywords": ["rosie", "riveter", "propaganda", "we can do it"],
    },
    {
        "id": "uncle-sam",
        "name": "Uncle Sam Wants You",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/1/1d/IWantYouPoster-MagazineCoverOrigVersion.jpg/800px-IWantYouPoster-MagazineCoverOrigVersion.jpg",
        "source": "https://commons.wikimedia.org/wiki/File:IWantYouPoster-MagazineCoverOrigVersion.jpg",
        "license": "Public domain (James Montgomery Flagg, 1917 — US work)",
        "keywords": ["uncle sam", "recruitment", "war"],
    },
    {
        "id": "keep-calm",
        "name": "Keep Calm and Carry On",
        "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/0/0b/Keep_Calm_and_Carry_On_Poster.svg/800px-Keep_Calm_and_Carry_On_Poster.svg.png",
        "source": "https://commons.wikimedia.org/wiki/File:Keep_Calm_and_Carry_On_Poster.svg",
        "license": "Public domain (UK Crown copyright expired / wartime poster)",
        "keywords": ["keep calm", "british", "poster"],
    },
    {
        "id": "alien-autopsy-still",
        "name": "Roswell Alien (Hoax Still)",
        "url": "https://upload.wikimedia.org/wikipedia/en/b/b7/Alien_autopsy.jpg",
        "source": "https://en.wikipedia.org/wiki/File:Alien_autopsy.jpg",
        "license": "Promotional/fair-use still from hoax film; use at own risk",
        "keywords": ["roswell", "alien", "ufo", "autopsy"],
    },
]

# Manually sourced "The Office" (NBC, 2005-2013) TV screencaps, hand-picked
# beyond what's in memegen's own catalog (which only has dwight/jim/
# michael-scott). Same legal footing as the memegen imports — fair-use
# parody screencaps, not a rights clearance — see NOTICE.md.
OFFICE = [
    {
        "id": "prison-mike",
        "name": "Prison Mike",
        "url": "https://i.imgflip.com/15okzz.jpg",
        "source": "https://imgflip.com/memegenerator/70011215/Prison-mike",
        "license": "Fair-use TV screencap (The Office, NBC, S03E09 \"The Convict\", 2006)",
        "keywords": ["the office", "prison mike", "michael scott"],
        "boxes": [
            {"name": "top", "x": 0.05, "y": 0.0, "width": 0.9, "height": 0.2,
             "align": "center", "valign": "top", "uppercase": True},
            {"name": "bottom", "x": 0.05, "y": 0.78, "width": 0.9, "height": 0.2,
             "align": "center", "valign": "bottom", "uppercase": True},
        ],
        "example_text": ["THEY SAID THE ONCALL ROTATION WAS FINE", "LET ME TELL YOU ABOUT PRISON"],
    },
    {
        "id": "kevins-chili",
        "name": "Kevin's Chili",
        "url": "https://i.imgflip.com/2ovv1u.jpg",
        "source": "https://imgflip.com/memetemplate/162729714/Kevins-Chili",
        "license": "Fair-use TV screencap (The Office, NBC, S06E01 \"Gossip\", 2009)",
        "keywords": ["the office", "kevin malone", "chili", "spill"],
        # Image is already split top/bottom into two panels; captions sit
        # near the top of each panel rather than at the image's extremes.
        "boxes": [
            {"name": "top", "x": 0.05, "y": 0.02, "width": 0.9, "height": 0.14,
             "align": "center", "valign": "top", "uppercase": True},
            {"name": "bottom", "x": 0.05, "y": 0.52, "width": 0.9, "height": 0.14,
             "align": "center", "valign": "top", "uppercase": True},
        ],
        "example_text": ["CARRYING FIVE YEARS OF TECH DEBT", "IT ALL HITS PRODUCTION AT ONCE"],
    },
    {
        "id": "creed",
        "name": "Creed Bratton",
        "url": "https://i.imgflip.com/1f905q.jpg",
        "source": "https://imgflip.com/memetemplate/86080526/Creed-The-Office",
        "license": "Fair-use TV screencap (The Office, NBC, 2005-2013)",
        "keywords": ["the office", "creed", "creed bratton", "creed thoughts"],
        "boxes": [
            {"name": "top", "x": 0.05, "y": 0.0, "width": 0.9, "height": 0.18,
             "align": "center", "valign": "top", "uppercase": True},
            {"name": "bottom", "x": 0.05, "y": 0.8, "width": 0.9, "height": 0.18,
             "align": "center", "valign": "bottom", "uppercase": True},
        ],
        "example_text": ["WROTE THE WHOLE THING IN COBOL", "NOBODY ASKED QUESTIONS"],
    },
]


def import_office(skip_download: bool) -> int:
    ok = 0
    for entry in OFFICE:
        tid = entry["id"]
        if tid in KEEP:
            continue
        dest_dir = TEMPLATES / tid
        img_path = dest_dir / "default.jpg"
        if not skip_download and not img_path.exists():
            print(f"  download office {tid}...")
            if not download(entry["url"], img_path):
                continue
        if not img_path.exists():
            continue
        cfg = {
            "name": entry["name"],
            "source": entry["source"],
            "license": entry["license"],
            "keywords": entry["keywords"],
            "background": {"image": "default.jpg"},
            "text_boxes": entry["boxes"],
            "example_text": entry["example_text"],
        }
        dest_dir.mkdir(parents=True, exist_ok=True)
        with open(dest_dir / "config.yaml", "w") as f:
            yaml.dump(cfg, f, default_flow_style=False, sort_keys=False)
        ok += 1
    return ok


def import_wikimedia(skip_download: bool) -> int:
    ok = 0
    for entry in WIKIMEDIA:
        tid = entry["id"]
        if tid in KEEP:
            continue
        dest_dir = TEMPLATES / tid
        img_path = dest_dir / "default.jpg"
        if not skip_download and not img_path.exists():
            print(f"  download wikimedia {tid}...")
            if not download(entry["url"], img_path):
                continue
        if not img_path.exists():
            continue
        cfg = {
            "name": entry["name"],
            "source": entry["source"],
            "license": entry["license"],
            "keywords": entry["keywords"],
            "background": {"image": "default.jpg"},
            "text_boxes": text_boxes(2),
            "example_text": [
                "WHEN THE SKY DOES THAT",
                "AND YOU'RE OUT OF 40MM",
            ],
        }
        dest_dir.mkdir(parents=True, exist_ok=True)
        with open(dest_dir / "config.yaml", "w") as f:
            yaml.dump(cfg, f, default_flow_style=False, sort_keys=False)
        ok += 1
    return ok


def load_custom_examples() -> dict[str, list[str]]:
    if not CUSTOM.exists():
        return {}
    with open(CUSTOM) as f:
        return yaml.safe_load(f) or {}


def patch_example_text(custom: dict[str, list[str]]) -> None:
    """Update example_text in configs where we have custom captions."""
    for tid, texts in custom.items():
        cfg_path = TEMPLATES / tid / "config.yaml"
        if not cfg_path.exists():
            continue
        with open(cfg_path) as f:
            cfg = yaml.safe_load(f)
        cfg["example_text"] = texts
        with open(cfg_path, "w") as f:
            yaml.dump(cfg, f, default_flow_style=False, sort_keys=False, allow_unicode=True)


def render_examples(custom: dict[str, list[str]], binary: Path) -> tuple[int, int]:
    ok, fail = 0, 0
    if not binary.exists():
        print("Building meme-cli...")
        subprocess.run(["make", "build"], cwd=ROOT, check=True)
    for cfg_path in sorted(TEMPLATES.glob("*/config.yaml")):
        tid = cfg_path.parent.name
        texts = custom.get(tid)
        if not texts:
            with open(cfg_path) as f:
                cfg = yaml.safe_load(f)
            texts = cfg.get("example_text") or []
        if not texts or all(not t.strip() for t in texts):
            continue
        out_dir = EXAMPLES / tid
        out_dir.mkdir(parents=True, exist_ok=True)
        out_path = out_dir / "eg01.png"
        cmd = [str(binary), "--meme-dir", str(TEMPLATES), "render", tid, *texts, "-o", str(out_path)]
        try:
            subprocess.run(cmd, cwd=ROOT, check=True, capture_output=True, text=True)
            print(f"  example {tid}/eg01.png")
            ok += 1
        except subprocess.CalledProcessError as e:
            print(f"  FAIL render {tid}: {e.stderr or e}")
            fail += 1
    return ok, fail


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--skip-download", action="store_true")
    parser.add_argument("--examples-only", action="store_true")
    parser.add_argument("--skip-memegen", action="store_true")
    args = parser.parse_args()

    custom = load_custom_examples()

    if not args.examples_only:
        if not args.skip_memegen:
            n, skipped = import_memegen(args.skip_download)
            print(f"Imported {n} memegen templates ({skipped} skipped)")
        nwiki = import_wikimedia(args.skip_download)
        print(f"Imported {nwiki} Wikimedia templates")
        noffice = import_office(args.skip_download)
        print(f"Imported {noffice} Office templates")

    patch_example_text(custom)
    print(f"Patched example_text on {len(custom)} templates")

    if not args.examples_only:
        print("Syncing text box geometry from memegen configs...")
        subprocess.run(
            [sys.executable, str(Path(__file__).parent / "sync_memegen_layouts.py")],
            cwd=ROOT,
            check=True,
        )

    binary = ROOT / "meme-cli"
    ok, fail = render_examples(custom, binary)
    print(f"Rendered {ok} examples ({fail} failed)")

    total = len(list(TEMPLATES.glob("*/config.yaml")))
    print(f"Total templates: {total}")


if __name__ == "__main__":
    main()
