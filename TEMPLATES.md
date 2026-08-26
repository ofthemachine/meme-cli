# Template library provenance

`meme-cli` ships a template library under `templates/`, embedded into the
binary and used whenever `MEME_DIR` isn't set. Every template records its
`source` and `license` in `config.yaml`. See also [`NOTICE.md`](NOTICE.md)
for the project-level attribution this table summarizes.

## What's in the library (~220 templates)

| Category | Count | License notes |
|---|---|---|
| Original seed (art / NASA) | 11 | Public domain — see table below |
| [jacebrowning/memegen](https://github.com/jacebrowning/memegen) imports | ~200 | Imported from memegen (MIT licensed). Each template's `source` field is a provenance pointer to the meme's origin (e.g. a Know Your Meme page) — the same disclosure memegen itself makes, not a per-image copyright clearance. Typical online meme use is parody/commentary; see each template's `source` for the underlying meme's own history. |
| Wikimedia / NASA extras | 1 (`starry-night`) | Public domain. Several more are listed in `scripts/bootstrap_templates.py`'s `WIKIMEDIA` table but currently fail to download — most of those upstream Wikimedia thumbnail URLs now 404/400; see the script for the full (currently-broken) list. |
| The Office (NBC) screencaps | 3 | Manually sourced beyond memegen's own catalog (which only has `dwight`/`jim`/`michael-scott`). Fair-use TV screencaps, same legal footing as the memegen imports — not a rights clearance. |

Run `meme-cli list` or `meme-cli search <query>` to browse. Layout coordinates
for memegen imports use a standard fractional top/bottom (or stacked) scheme —
good enough for most captions; tweak individual `config.yaml` files if a
template needs tighter positioning.

### Original public-domain seed set

| id | license |
|---|---|
| `quote-card` | Original to meme-cli; no third-party image |
| `the-scream` | Public domain (Edvard Munch, 1893) |
| `mona-lisa` | Public domain (Leonardo da Vinci, c. 1503) |
| `great-wave` | Public domain (Katsushika Hokusai, c. 1831) |
| `creation-of-adam` | Public domain (Michelangelo, c. 1512) |
| `vitruvian-man` | **CC BY-SA 3.0** — photo by Luc Viatour, attribution required if redistributed; underlying drawing is public domain |
| `earthrise` | Public domain (NASA/Bill Anders, Apollo 8, 1968 — US government work) |
| `moon-landing` | Public domain (NASA/Neil Armstrong, Apollo 11, 1969 — US government work) |
| `pearl-earring` | Public domain (Johannes Vermeer, c. 1665) |
| `napoleon-alps` | Public domain (Jacques-Louis David, 1801) |
| `liberty-leading` | Public domain (Eugène Delacroix, 1830) |

### The Office (NBC) screencaps

Beyond memegen's own catalog (`dwight`, `jim`, `michael-scott`), a few more
were sourced directly for this library:

| id | license |
|---|---|
| `prison-mike` | Fair-use TV screencap (S03E09 "The Convict", 2006) |
| `kevins-chili` | Fair-use TV screencap (S06E01 "Gossip", 2009) |
| `creed` | Fair-use TV screencap (2005-2013) |

## Examples (local only)

Rendered sample memes live in `examples/<template-id>/eg01.png` — **gitignored**.
One example per template. Regenerate with:

```bash
python3 scripts/bootstrap_templates.py --examples-only
python3 scripts/generate_gallery.py   # or: make gallery
open examples/gallery.html            # review all 200+ memes in one page
```

Custom captions (Foxhole / UFO / tech-nerd themed) are in
`scripts/custom_examples.yaml`. Examples aim to use all text boxes when a
template supports multiple lines.

## Adding more templates

1. Create a directory named for the template's id.
2. Drop a background image (`default.jpg` or `.png`), or use a flat
   `background.color` card (see `quote-card`).
3. Write `config.yaml` — either an axis-aligned box (`x`, `y`, `width`,
   `height`, optional `angle`) **or** a `quad` write face: four corners
   `[tl, tr, br, bl]`. Three points `[tl, tr, bl]` are a wrapper that implies
   `br` as a parallelogram. Text is laid out orthogonally then projectively
   mapped onto the quad (perspective-capable). Legacy key `parallelogram`
   is accepted as an alias for `quad`.
4. Record `source` and `license` honestly.
5. `meme-cli show <id>` and `meme-cli render <id> ...` to sanity-check.

To bulk-import from memegen.link:

```bash
python3 scripts/bootstrap_templates.py          # download + write configs
python3 scripts/sync_memegen_layouts.py         # fix text box geometry from memegen
python3 scripts/bootstrap_templates.py --examples-only  # re-render examples
```

Imported memegen templates get **precise text box coordinates** converted from
memegen's `anchor_x`/`scale_x` schema (via `sync_memegen_layouts.py`), including
`angle` when a caption is rotated. Templates with complex multi-panel layouts
(e.g. Gandalf, Bilbo) may need manual tuning — see the sync script's skip list.

For slanted / perspective write faces, mark corners visually:

```bash
python3 scripts/parallelogram_editor.py --open   # or ?t=cmm
```

Click **TL → TR → BR → BL** (or 3-pt mode), copy the `quad:` YAML into
`config.yaml`.

Good hunting grounds: Wikimedia Commons (public-domain/CC filter), NASA/ESA
archives, and [jacebrowning/memegen](https://github.com/jacebrowning/memegen)
itself, which is where the bulk of this library's templates already come
from (see [`NOTICE.md`](NOTICE.md)).
