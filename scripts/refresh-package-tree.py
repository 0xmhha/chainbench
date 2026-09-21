#!/usr/bin/env python3
"""Rewrite every figure in docs/dev/architecture/package-tree.md from the tree.

internal/arch measures that document three ways -- the §0 totals, each
per-package figure, and each section heading against what its own section lists
-- so any change to the code moves numbers a person would otherwise hunt. This
walks the same way the ratchet does and writes the answers back, preserving the
column the numbers are aligned on.

Run it from the repository root after a change that moves lines:

    python3 scripts/refresh-package-tree.py

It is the counterpart to the check, not a second source of truth: if the two
ever disagree, the ratchet is right and this is wrong.
"""
import re,io,os,sys
real={}
for base in ("internal","cmd","scripts"):
    for root,_,fs in os.walk(base):
        gos=[f for f in fs if f.endswith(".go") and not f.endswith("_test.go")]
        if gos:
            real[root]=sum(sum(1 for _ in io.open(os.path.join(root,f),encoding="utf-8",errors="replace")) for f in gos)
p="docs/dev/architecture/package-tree.md"
doc=io.open(p,encoding="utf-8").read().split("\n")
entry=re.compile(r"^((?:[│ ]*[├└]── )?)(\(?[a-z][a-z0-9/._-]*\)?)(\s+)([\d,]+)(\s)")
rootline=re.compile(r"^([a-z][a-z0-9/]*/)(\s|$)")
inb=False;cur=None;sec=None;bysec={};fixed=0
for i,l in enumerate(doc):
    if l.startswith("## "): sec=l.strip(); bysec.setdefault(sec,[])
    if l.startswith("```"): inb=not inb; cur=None; continue
    if not inb: continue
    m=rootline.match(l)
    if m: cur=m.group(1).rstrip("/"); continue
    m=entry.match(l)
    if not m: continue
    pre,name,gap,num,tail=m.groups()
    path = cur if name.startswith("(") else (name if name.startswith(("internal/","cmd/","scripts/")) else (cur+"/"+name if cur else name))
    if path not in real:
        c=[k for k in real if k.endswith("/"+name) and (cur is None or k.startswith(cur))]
        if len(c)==1: path=c[0]
    if path not in real: continue
    bysec[sec].append(path)
    want=f"{real[path]:,}"
    if want!=num:
        d=len(want)-len(num); g=gap[:-d] if d>0 else gap+" "*(-d)
        doc[i]=pre+name+(g or " ")+want+tail+l[m.end():]; fixed+=1
secre=re.compile(r"^(## \d+\. .*?)(\d+)(패키지 )([\d,]+)(줄)")
for i,l in enumerate(doc):
    m=secre.match(l)
    if not m: continue
    ps=bysec.get(l.strip(),[])
    if not ps: continue
    doc[i]=f"{m.group(1)}{len(ps)}{m.group(3)}{sum(real[x] for x in ps):,}{m.group(5)}"+l[m.end():]
tot={"internal/":0,"cmd/":0,"scripts/inventory/":0}
cnt={k:0 for k in tot}
for k,v in real.items():
    for pre in tot:
        if (k+"/").startswith(pre): tot[pre]+=v; cnt[pre]+=1
for i,l in enumerate(doc):
    for pre in tot:
        if l.startswith(f"| `{pre}` |"):
            doc[i]=f"| `{pre}` | {cnt[pre]} | {tot[pre]:,} |"
    if l.startswith("| **합계**"):
        doc[i]=f"| **합계** | **{sum(cnt.values())}** | **{sum(tot.values()):,}** |"
io.open(p,"w",encoding="utf-8").write("\n".join(doc))
print(f"  패키지 {fixed}개 · 절 제목 · §0 표 갱신")
