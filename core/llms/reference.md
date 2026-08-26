meme-cli — reference for AI agents

meme-cli composites caption text onto meme template backgrounds, memegen.link
style. This is a full command/data reference — enough to explore the tool
and, if needed, write your own guide or skill for it — not a tutorial.

## Commands

- `meme-cli search <query>` — find templates by id/name/keyword substring
  match. Output: one `<id>  <name>` line per match. Cheapest way to browse.
- `meme-cli show <template-id> [--json]` — print a template's metadata and
  resolved text-box layout: name, source, license, keywords, each box's
  index/name/geometry (in the order render fills them), and its
  example_text. With `--json`, prints `{"meme_dir": ..., "template": {...}}`
  — the structured equivalent, keyed the same way as one entry of
  `list --json`'s `templates` map.
- `meme-cli render <template-id> [text...] [-o out.png]` — composite the
  given captions onto the template. Args after the template id fill its text
  boxes positionally, in the order `show` lists them. Omit all text args to
  render the template's example_text instead. `-o`/`--out` sets the output
  path; format is chosen from its extension (.png default, .jpg/.jpeg).
  `--debug-boxes` overlays magenta outlines of each box for layout QA.
- `meme-cli list [--json]` — every template's id + name (or, with --json,
  `{"meme_dir": ..., "templates": {id: full metadata}}`: name, source,
  license, keywords, text_boxes, example_text). Expensive relative to
  `search`; prefer search unless you need a full dump.
- `meme-cli version` — print the build version.
- `meme-cli llms` — this reference.

Global flag: `--meme-dir <dir>` (or `$MEME_DIR` env var) points at a
directory of templates on disk instead of the library bundled in the
binary — flag wins over env, env wins over the bundled default. This is the
mechanism for running meme-cli against a custom/mounted template library,
e.g. in a container. `list`/`show` (both human and `--json` output) report
which one is actually in effect as `meme_dir` / "meme dir: ...".

## Exit codes

`0` success. `2` the template id or `--meme-dir` path doesn't exist —
distinguish this from a broken/invalid template (bad config.yaml, render
failure) or bad CLI usage, which use `1`. Useful for an agent deciding
whether to retry with a different template id vs. treat the template as
broken.

## Text rendering conventions

Captions are auto-uppercased and auto-sized/word-wrapped to fit their box
(shrinking font size, then still rendering — never erroring — if a caption
can't fit even at the minimum size). Don't hand-wrap or hand-case caption
text; pass it as plain phrases.

## Template data model

Each template is a directory (its id) with a `config.yaml`: `name`,
`source` (provenance URL), `license`, `keywords`, a `background` (an image
file, or a flat `color`+`width`+`height`), a list of `text_boxes` (each with
fractional `x`/`y`/`width`/`height`, optional `angle`, or a `quad` for
perspective/slanted write faces), and `example_text`. `show <id>` renders
this in human-readable form; `list --json`/`show` are the two ways to
introspect it without reading YAML directly.

## Suggested discovery workflow

1. `search <keyword(s)>` for the situation (try a few keywords).
2. `show <template-id>` on the best match to see box count/order.
3. `render <template-id> "<box 1 text>" "<box 2 text>" ...`.
