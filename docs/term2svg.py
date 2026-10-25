#!/usr/bin/env python3
"""Render captured terminal output as a terminal-window SVG screenshot.

Usage:
    ./zt-sim policies | python3 docs/term2svg.py "zt-sim policies" > docs/policies.svg

Regenerate every screenshot the README shows: make screenshots
(python3 is doc tooling only — the zt-sim binary stays stdlib-only Go).
"""
import html
import re
import sys

# Catppuccin Mocha-ish terminal palette.
BG, FG, DIM, SEP = "#1e1e2e", "#cdd6f4", "#7f849c", "#313244"
HEADER, PREFIX, PASS = "#89b4fa", "#94e2d5", "#a6e3a1"
ACTION = {"BLOCK": "#f38ba8", "STEPUPMFA": "#f9e2af",
          "LIMITED": "#cba6f7", "ALLOW": "#a6e3a1"}
TOKEN = re.compile(r"Allow|Limited|StepUpMFA|Block|ALLOW|LIMITED|STEPUPMFA|BLOCK")

FONT, CHAR_W, LINE_H = 13, 7.8, 21
PAD_L, PAD_R, HEAD_H = 22, 22, 38
DOTS = [("#ff5f57", 24), ("#febc2e", 46), ("#28c840", 68)]

WHOLE_LINE = [  # (regex, color) — first match wins
    (re.compile(r"^PASS "), PASS),
    (re.compile(r"^(USER|PACK|ID) "), HEADER),
    (re.compile(r"^(usage:|pillars:|audit:|counts:)"), PREFIX),
]


def esc(s):
    return html.escape(s, quote=False)


def segments(line):
    """Split a line into (text, color|None) segments; None inherits FG."""
    for rx, color in WHOLE_LINE:
        if rx.search(line):
            return [(line, color)]
    segs, pos = [], 0
    for m in TOKEN.finditer(line):
        if m.start() > pos:
            segs.append((line[pos:m.start()], None))
        segs.append((m.group(), ACTION.get(m.group().upper())))
        pos = m.end()
    if pos < len(line):
        segs.append((line[pos:], None))
    return segs or [("", None)]


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: term2svg.py WINDOW-TITLE  (input on stdin)")
    lines = [l.rstrip("\r") for l in sys.stdin.read().splitlines()]
    if not lines or not any(lines):
        sys.exit("term2svg.py: no input on stdin")

    width = PAD_L + int(max(map(len, lines)) * CHAR_W + 0.5) + PAD_R
    height = HEAD_H + LINE_H * len(lines) + 18
    cx = width // 2

    out = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" '
        f'viewBox="0 0 {width} {height}">',
        f'<rect width="{width}" height="{height}" rx="10" fill="{BG}"/>',
        f'<line x1="0" y1="{HEAD_H}" x2="{width}" y2="{HEAD_H}" stroke="{SEP}"/>',
    ]
    for color, x in DOTS:
        out.append(f'<circle cx="{x}" cy="{HEAD_H // 2}" r="5.5" fill="{color}"/>')
    out.append(
        f'<text x="{cx}" y="{HEAD_H // 2 + 4}" text-anchor="middle" '
        f'font-family="ui-sans-serif, system-ui, sans-serif" font-size="12" '
        f'fill="{DIM}">{esc(sys.argv[1])}</text>'
    )
    out.append(
        f'<g font-family="ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" '
        f'font-size="{FONT}" fill="{FG}">'
    )
    y = HEAD_H + 27
    for line in lines:
        spans = "".join(
            f'<tspan fill="{color}">{esc(text)}</tspan>' if color else f"<tspan>{esc(text)}</tspan>"
            for text, color in segments(line)
        )
        out.append(f'<text x="{PAD_L}" y="{y}" xml:space="preserve">{spans}</text>')
        y += LINE_H
    out.append("</g>\n</svg>")
    print("\n".join(out))


if __name__ == "__main__":
    main()
