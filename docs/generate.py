#!/usr/bin/env python3
"""
generate.py - build a Google-Docs-ready .docx from a Markdown file.

Markdown here uses Mermaid diagrams, which pandoc cannot render on its own.
This script pre-renders each ```mermaid block to a PNG with mermaid-cli, swaps
the block for an image reference, then runs pandoc to produce the .docx.

Layout (relative to this script, which lives in docs/):

    docs/md/<name>.md      source Markdown          (input)
    docs/images/<name>-N.png  rendered diagrams      (generated)
    docs/<name>.docx       final document           (output)

Usage:

    ./generate.py                       # build every .md in docs/md/
    ./generate.py Per-Key-Loraraw       # build just docs/md/Per-Key-Loraraw.md

Requires: pandoc, and mmdc (mermaid-cli, `npm i -g @mermaid-js/mermaid-cli`).
"""

import os
import re
import subprocess
import sys

DOCS = os.path.dirname(os.path.abspath(__file__))
MD_DIR = os.path.join(DOCS, "md")
IMG_DIR = os.path.join(DOCS, "images")

MERMAID_RE = re.compile(r"```mermaid\n(.*?)```", re.S)


def need(tool):
    if subprocess.run(["which", tool], capture_output=True).returncode != 0:
        sys.exit("missing tool: %s" % tool)


def render_mermaid(block, out_png):
    """Render one mermaid block to a PNG. Returns True on success."""
    src = out_png + ".mmd"
    with open(src, "w", encoding="utf-8") as f:
        f.write(block)
    # White background and 2x scale so it stays readable inside a document.
    r = subprocess.run(
        ["mmdc", "-i", src, "-o", out_png, "-b", "white", "-s", "2", "-q"],
        capture_output=True, text=True, timeout=180,
    )
    os.remove(src)
    if r.returncode != 0:
        sys.stderr.write((r.stderr or r.stdout) + "\n")
        return False
    return True


def build(name):
    md_path = os.path.join(MD_DIR, name + ".md")
    if not os.path.isfile(md_path):
        sys.exit("not found: %s" % md_path)

    os.makedirs(IMG_DIR, exist_ok=True)
    text = open(md_path, encoding="utf-8").read()

    # Replace each mermaid block with an image, in document order.
    count = [0]

    def repl(m):
        count[0] += 1
        n = count[0]
        png = os.path.join(IMG_DIR, "%s-%02d.png" % (name, n))
        if not render_mermaid(m.group(1), png):
            sys.exit("mermaid render failed for diagram %d in %s" % (n, name))
        # Path is relative to the .docx, which sits in docs/.
        rel = os.path.join("images", os.path.basename(png))
        print("  diagram %2d -> %s" % (n, rel))
        return "![](%s)\n" % rel

    processed = MERMAID_RE.sub(repl, text)

    # Hand the rewritten Markdown to pandoc on stdin. resource-path lets it find
    # the images relative to docs/.
    docx = os.path.join(DOCS, name + ".docx")
    r = subprocess.run(
        ["pandoc", "-f", "gfm", "-t", "docx",
         "--resource-path", DOCS,
         "--toc", "--toc-depth=2",
         "-o", docx, "-"],
        input=processed, capture_output=True, text=True,
    )
    if r.returncode != 0:
        sys.exit("pandoc failed: %s" % (r.stderr or r.stdout))

    size = os.path.getsize(docx)
    print("built %s (%d diagrams, %d KB)" % (docx, count[0], size // 1024))


def main():
    need("pandoc")
    need("mmdc")
    if len(sys.argv) > 1:
        names = [os.path.splitext(os.path.basename(a))[0] for a in sys.argv[1:]]
    else:
        names = sorted(
            os.path.splitext(f)[0]
            for f in os.listdir(MD_DIR)
            if f.endswith(".md")
        )
    for name in names:
        print("building %s.md ..." % name)
        build(name)


if __name__ == "__main__":
    main()
