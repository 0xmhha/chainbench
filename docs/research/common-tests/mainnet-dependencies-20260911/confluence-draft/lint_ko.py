#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""초안 문구 점검: 개발 내부 용어, 분석 내부 번호, 코드 경로, 번역투 표현을 찾는다."""
import re,glob,sys
BAN=[r'\badapter\b',r'\boracle\b',r'config\+oracle',r'config-only',r'\bfixture\b',r'\bsentinel\b',r'\$binding',r'\bbinding\b',r'\bCD-[AB]-\d',r'\bH-\d{2}\b',r'\.go\b',r'\bgate\b',r'\bcapability\b',r'\bpreflight\b',r'\bharness\b',r'\bcompose\b',r'\battach\b',r'\bEnvV2\b',r'\bDSL\b',r'\bfork\b',r'\bprofile\b',r'\bnamespace\b',r'\bbaseFee\b',r'\bfeeCap\b',r'\bgasTip\b',r'\bistanbul_',r'\bwemix_',r'\beth_[a-zA-Z]+',r'\btxpool_',r'\badmin_',r'\bJSON\b',r'\bAPI 묶음\b(?!.*뜻)',r'핵심은',r'~을 열었',r'에 대한',r'를 통해',r'것이 가능',r'—']
ALLOW_FILES={'20-glossary.md'}  # 용어집은 원어 병기를 허용
for f in sorted(glob.glob('*.md')):
    hits=[]
    for ln,line in enumerate(open(f,encoding='utf-8'),1):
        for b in BAN:
            for m in re.finditer(b,line):
                if f in ALLOW_FILES and b in (r'\beth_[a-zA-Z]+',r'\bistanbul_',r'\btxpool_',r'\badmin_'): continue
                hits.append((ln,m.group(0),line.strip()[:90]))
    print(f'== {f}: {len(hits)}')
    for h in hits[:12]: print('  ',h)
