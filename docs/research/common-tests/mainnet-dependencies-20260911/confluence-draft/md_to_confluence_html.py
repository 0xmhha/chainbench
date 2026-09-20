"""Convert the confluence-draft Markdown pages into Confluence HTML with
content-proportional table column widths.

Only the Markdown subset used in the drafts is handled: headings, paragraphs,
bullet and numbered lists (one level), tables, bold, inline code, links.
Column widths are derived from display width (CJK = 2 units) so that long
columns get more room and short columns (IDs, priorities) stop wrapping.
"""
from __future__ import annotations

import html
import re
import sys
import unicodedata
from pathlib import Path

PAGE_URLS = {
    "공통 테스트 목록": "https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2988965889",
    "메인넷별 의존 요소": "https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2988376101",
    "별도 구현이 필요한 항목": "https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987884735",
    "공통 테스트 분리 변경 범위": "https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987720853",
    "회귀 실행 묶음": "https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2986934428",
    "상세 실행 절차": "https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987196682",
    "용어집": "https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2986868908",
    "부록 A. 두 체인에서만 가능한 테스트": "https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2986901809",
    "부록 B. 공통에서 제외한 테스트와 이유": "https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987524189",
}

TOTAL_WIDTH_MAX = 1600  # px, page width is "max"
TOTAL_WIDTH_MIN = 700
PX_PER_UNIT = 8.5       # px per latin char unit at 14px font
CELL_PAD = 22           # cell padding + border, px
SHORT_UNITS = 36        # columns whose longest value fits here never wrap
CAP_UNITS = 120         # a single very long cell must not eat the whole table
MIN_PROSE_PX = 150
FULL_WIDTH_FROM_PX = 1100  # tables at least this wide stretch to the page width


def disp_width(text: str) -> int:
    """Display width in latin-char units; CJK counts as 2."""
    w = 0
    for ch in text:
        if unicodedata.east_asian_width(ch) in ("W", "F"):
            w += 2
        else:
            w += 1
    return w


def inline(text: str) -> str:
    """Markdown inline -> HTML: bold, code, links, bracketed page names."""
    out = []
    pos = 0
    pattern = re.compile(r"(\*\*(.+?)\*\*)|(`([^`]+)`)|(\[([^\]]+)\]\(([^)]+)\))|(\[([^\]]+)\])")
    for m in pattern.finditer(text):
        out.append(html.escape(text[pos:m.start()], quote=False))
        if m.group(1):
            out.append(f"<strong>{inline(m.group(2))}</strong>")
        elif m.group(3):
            out.append(f"<code>{html.escape(m.group(4), quote=False)}</code>")
        elif m.group(5):
            out.append(f'<a href="{html.escape(m.group(7), quote=True)}">{html.escape(m.group(6), quote=False)}</a>')
        else:
            name = m.group(9)
            if name in PAGE_URLS:
                out.append(f'<a href="{PAGE_URLS[name]}">{html.escape(name, quote=False)}</a>')
            else:
                out.append(html.escape(m.group(0), quote=False))
        pos = m.end()
    out.append(html.escape(text[pos:], quote=False))
    return "".join(out)


def strip_inline(text: str) -> str:
    text = re.sub(r"\*\*(.+?)\*\*", r"\1", text)
    text = re.sub(r"`([^`]+)`", r"\1", text)
    text = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", text)
    return text


def split_row(line: str) -> list[str]:
    line = line.strip()
    if line.startswith("|"):
        line = line[1:]
    if line.endswith("|"):
        line = line[:-1]
    return [c.strip() for c in line.split("|")]


def col_widths(rows: list[list[str]]) -> list[int]:
    """Short columns keep their natural width; long prose columns share the rest."""
    ncol = max(len(r) for r in rows)
    longest, typical = [], []
    for c in range(ncol):
        ws = sorted(disp_width(strip_inline(r[c])) for r in rows if c < len(r))
        longest.append(max(ws))
        typical.append(max(ws[int(len(ws) * 0.75)], disp_width(strip_inline(rows[0][c])) + 2))
    fixed = {c: int(longest[c] * PX_PER_UNIT) + CELL_PAD for c in range(ncol) if longest[c] <= SHORT_UNITS}
    natural = sum(fixed.values()) + sum(min(typical[c], CAP_UNITS) * PX_PER_UNIT + CELL_PAD for c in range(ncol) if c not in fixed)
    total = max(TOTAL_WIDTH_MIN, min(TOTAL_WIDTH_MAX, natural))
    rest = total - sum(fixed.values())
    prose = [c for c in range(ncol) if c not in fixed]
    weights = {c: min(typical[c], CAP_UNITS) for c in prose}
    widths = [0] * ncol
    for c in range(ncol):
        widths[c] = fixed[c] if c in fixed else max(MIN_PROSE_PX, int(rest * weights[c] / sum(weights.values())))
    return widths


def render_table(lines: list[str]) -> str:
    rows = [split_row(l) for l in lines if not re.match(r"^\s*\|?\s*:?-{2,}", l)]
    widths = col_widths(rows)
    align_line = next((l for l in lines if re.match(r"^\s*\|?\s*:?-{2,}", l)), "")
    aligns = []
    for cell in split_row(align_line):
        aligns.append("end" if cell.endswith(":") and not cell.startswith(":") else "")

    def cell(tag: str, i: int, text: str) -> str:
        style = f' style="text-align: {aligns[i]}"' if i < len(aligns) and aligns[i] else ""
        return f'<{tag} data-colwidth="{widths[i]}"><p{style}>{inline(text)}</p></{tag}>'

    head = "".join(cell("th", i, t) for i, t in enumerate(rows[0]))
    body = []
    for r in rows[1:]:
        r = r + [""] * (len(widths) - len(r))
        body.append("<tr>" + "".join(cell("td", i, t) for i, t in enumerate(r)) + "</tr>")
    # Wide tables stretch to the full page width with columns scaled in
    # proportion (no fixed display mode); small tables keep their natural size.
    layout = "full-width" if sum(widths) >= FULL_WIDTH_FROM_PX else "default"
    return (
        f'<table data-layout="{layout}" data-width="{sum(widths)}">'
        f"<thead><tr>{head}</tr></thead><tbody>{''.join(body)}</tbody></table>"
    )


def render(md: str, drop_title: bool = True) -> str:
    lines = md.splitlines()
    out: list[str] = []
    i = 0
    if drop_title and lines and lines[0].startswith("# "):
        i = 1
    while i < len(lines):
        line = lines[i]
        if not line.strip():
            i += 1
            continue
        if line.startswith("|"):
            block = []
            while i < len(lines) and lines[i].startswith("|"):
                block.append(lines[i])
                i += 1
            out.append(render_table(block))
            continue
        m = re.match(r"^(#{1,6})\s+(.*)", line)
        if m:
            level = len(m.group(1))
            out.append(f"<h{level}>{inline(m.group(2))}</h{level}>")
            i += 1
            continue
        if re.match(r"^- ", line):
            items = []
            while i < len(lines) and re.match(r"^- ", lines[i]):
                items.append(f"<li><p>{inline(lines[i][2:])}</p></li>")
                i += 1
            out.append("<ul>" + "".join(items) + "</ul>")
            continue
        if re.match(r"^\d+\. ", line):
            items = []
            while i < len(lines) and re.match(r"^\d+\. ", lines[i]):
                items.append(f"<li><p>{inline(re.sub(r'^\d+\. ', '', lines[i]))}</p></li>")
                i += 1
            out.append('<ol start="1">' + "".join(items) + "</ol>")
            continue
        # paragraph: join consecutive plain lines
        para = []
        while i < len(lines) and lines[i].strip() and not lines[i].startswith(("|", "#", "- ")) and not re.match(r"^\d+\. ", lines[i]):
            para.append(lines[i].strip())
            i += 1
        out.append(f"<p>{inline(' '.join(para))}</p>")
    return "".join(out)


def main(argv: list[str]) -> int:
    src = Path(argv[1])
    dst = Path(argv[2])
    dst.write_text(render(src.read_text(encoding="utf-8")), encoding="utf-8")
    print(f"{src.name}: {dst.stat().st_size} bytes")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
