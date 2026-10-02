#!/usr/bin/env python3
"""tests/tc 의 케이스마다 복사해 붙일 명령을 만들어 HOW-TO-USE.md 의 목록 절을 채운다.

손으로 적으면 반드시 낡는다. 이 저장소가 그것을 한 번 겪었다 — RUN-EACH.md 가
"209건" 이라고 적은 채로 남아 있었고 실제는 197건이었다. 목록은 트리에서 만든다.

    python3 scripts/gen-case-commands.py          # 목록을 표준출력으로
    python3 scripts/gen-case-commands.py --write  # HOW-TO-USE.md 의 표시 구간을 갈아 끼운다

표시 구간은 아래 두 줄 사이다. 두 줄은 건드리지 않는다.

    <!-- BEGIN generated: scripts/gen-case-commands.py -->
    <!-- END generated -->
"""
import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
DOC = ROOT / "tests/tc/HOW-TO-USE.md"
BEGIN = "<!-- BEGIN generated: scripts/gen-case-commands.py -->"
END = "<!-- END generated -->"

# 체인마다 바이너리 변수 이름이 다르다. 문서 앞머리가 이 셋을 export 하게 한다.
BINVAR = {"stablenet": "$GSTABLE", "wbft": "$GWBFT", "wemix": "$GWEMIX"}

# poa 는 노드마다 p2p 옆 포트를 셋 잡아 서버 세트가 다르고, 합류가 느려 대기를 늘린다.
DOCKER_COMMON = (
    "--server-set env/docker/build/server-set.yaml "
    "--workspace-config env/docker/build/workspace-config.yaml "
    "--docker --all-servers --keys-source generate"
)
DOCKER_WEMIX = (
    "--server-set env/docker/build/server-set-wemix.yaml "
    "--workspace-config env/docker/build/workspace-config.yaml "
    "--docker --all-servers --keys-source generate --node-monitor-timeout 5m"
)


def preset_of(spec: dict) -> dict:
    """케이스가 부르는 chain-preset 을 끝까지 따라가 펼친다."""
    cp, seen = spec.get("chainPreset"), set()
    while True:
        if isinstance(cp, dict):
            ext = cp.get("extends")
            if not ext or ext in seen:
                return cp
            seen.add(ext)
            base = preset_of({"chainPreset": ext})
            base.update({k: v for k, v in cp.items() if k != "extends"})
            return base
        if isinstance(cp, str):
            if cp in seen:
                return {}
            seen.add(cp)
            f = ROOT / "presets/chain" / (cp + ".json")
            if not f.exists():
                return {}
            cp = json.loads(f.read_text())
            continue
        return {}


def envs_of(preset: dict) -> list[str]:
    """프리셋이 요구하는 환경변수 중 체인 바이너리가 아닌 것."""
    names = set(re.findall(r"\$\{([A-Z_]+)(?::-[^}]*)?\}", json.dumps(preset)))
    return sorted(n for n in names if not n.endswith("_BIN") or n.startswith("GSTABLE_"))


def rows():
    for f in sorted((ROOT / "tests/tc").rglob("*.json")):
        spec = json.loads(f.read_text())
        rel = f.relative_to(ROOT).as_posix()
        p = preset_of(spec)
        chain = p.get("chain", "?")
        yield {
            "path": rel,
            "id": spec.get("id", ""),
            "chain": chain,
            "binvar": BINVAR.get(chain, "$GSTABLE"),
            "attach": bool(p.get("attach")),
            "remote": "target:remote" in (spec.get("requires") or []),
            "envs": [e for e in envs_of(p) if e not in ("GSTABLE_BIN",)],
            "ws": "~/cbw/one/" + f.stem,
        }


def render() -> str:
    out = []
    by_dir: dict[str, list] = {}
    for r in rows():
        by_dir.setdefault(str(pathlib.Path(r["path"]).parent), []).append(r)

    out.append(f"케이스 **{sum(len(v) for v in by_dir.values())}건**이다. "
               "각 케이스에 두 갈래를 적는다 — `bin/chainbench` 로 한 건만 돌리는 것과, "
               "`scripts/tcsweep.sh` 로 같은 한 건을 돌리는 것이다. 스크립트 쪽은 망을 "
               "세우고 내리고 지우는 것까지 하고 판정 한 줄을 남긴다.\n")

    for d in sorted(by_dir):
        rs = by_dir[d]
        out.append(f"### `{d.removeprefix('tests/tc/')}` — {len(rs)}건\n")
        for r in rs:
            name = pathlib.Path(r["path"]).name
            out.append(f"**{name}**" + (f" · `{r['id']}`" if r["id"] else ""))
            pre = "".join(f"{e}=<값> " for e in r["envs"])
            if r["attach"]:
                out.append("```sh\n"
                           f"{pre}bin/chainbench run {r['path']}\n"
                           "```\n"
                           "> 이미 떠 있는 망에 붙는다. 망을 세우지 않으므로 "
                           "`--workspace-dir` 도 `--binary` 도 주지 않는다. "
                           "`tcsweep.sh` 는 이 갈래를 자기가 세운 망에 붙여 돌린다.\n")
                continue
            if r["remote"]:
                flags = DOCKER_WEMIX if r["chain"] == "wemix" else DOCKER_COMMON
                out.append("```sh\n"
                           f"# 로컬에서는 건너뛴다 — 노드가 도는 기계의 셸이 필요하다\n"
                           f"bin/chainbench run {r['path']} \\\n"
                           f"  --workspace-dir {r['ws']} {flags}\n"
                           "```\n")
                continue
            out.append("```sh\n"
                       f"{pre}bin/chainbench run {r['path']} \\\n"
                       f"  --workspace-dir {r['ws']} --binary {r['binvar']}\n"
                       f"\n"
                       f"scripts/tcsweep.sh ~/cbw/one.log '{name[:-5]}'\n"
                       "```\n")
    return "\n".join(out)


def main() -> int:
    body = render()
    if "--write" not in sys.argv:
        print(body)
        return 0
    doc = DOC.read_text()
    if BEGIN not in doc or END not in doc:
        print(f"{DOC} 에 표시 구간이 없다: {BEGIN} / {END}", file=sys.stderr)
        return 2
    head, rest = doc.split(BEGIN, 1)
    _, tail = rest.split(END, 1)
    DOC.write_text(f"{head}{BEGIN}\n\n{body}\n{END}{tail}")
    print(f"{DOC} 갱신")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
