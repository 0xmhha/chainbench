---
name: chainbench 기관 일지
description: 체인 시험을 당직 일지처럼 적고, 실패를 기록계 띠 위에서 측정하는 대시보드
colors:
  ledger-stock: "#e6ece4"
  ledger-stock-raised: "#f5f8f3"
  ledger-ruling: "#7f9c88"
  ledger-ruling-faint: "#b9cbbd"
  ink: "#18201b"
  ink-muted: "#435048"
  ink-faint: "#56625a"
  blue-black-ink: "#23356e"
  red-ink: "#a8221a"
  green-ink: "#1d6537"
  ochre-ink: "#7c5200"
  gap-fill: "#d2d9cf"
  night-stock: "#131a15"
  night-ruling: "#557260"
  night-ink: "#e3eae1"
  night-red-ink: "#f2897f"
  night-blue-ink: "#a3b5ee"
typography:
  display:
    fontFamily: "Avenir Next, Segoe UI, Apple SD Gothic Neo, Noto Sans KR, sans-serif"
    fontSize: "30px"
    fontWeight: 700
    lineHeight: 1.2
  title:
    fontFamily: "Avenir Next Condensed, Roboto Condensed, Arial Narrow, Apple SD Gothic Neo, sans-serif"
    fontSize: "14px"
    fontWeight: 700
    letterSpacing: "0.02em"
  body:
    fontFamily: "Avenir Next, Segoe UI, Apple SD Gothic Neo, Noto Sans KR, sans-serif"
    fontSize: "14px"
    lineHeight: 1.5
  label:
    fontFamily: "Avenir Next Condensed, Roboto Condensed, Arial Narrow, Apple SD Gothic Neo, sans-serif"
    fontSize: "12px"
    fontWeight: 600
    letterSpacing: "0.04em"
  data:
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "13px"
rounded:
  none: "0"
spacing:
  1: "4px"
  2: "8px"
  3: "12px"
  4: "16px"
  5: "24px"
components:
  button:
    backgroundColor: "{colors.ledger-stock}"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    height: "36px"
  button-primary:
    backgroundColor: "{colors.blue-black-ink}"
    textColor: "{colors.ledger-stock-raised}"
    rounded: "{rounded.none}"
  input:
    backgroundColor: "{colors.ledger-stock-raised}"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    height: "36px"
---

# chainbench 기관 일지

## Overview

화면 하나하나가 선박 기관실의 당직 일지 한 쪽이다. 작업은 쪽 맨 위의 머리줄(folio)로 시작하고, 실패는 그 쪽 첫 줄에 빨간 잉크로 쓴 사고 기록이 된다. 그 아래에는 로직 분석기에서 빌려 온 기록계 띠가 놓인다. 테스트, 노드의 경고와 오류, 블록 높이, 피어, 대기 트랜잭션, 다른 노드의 오류를 한 시간축에 쌓고, 두 커서 A와 B 사이를 측정한다. 띠는 형광 화면이 아니라 장부의 괘선과 잉크로 그린다. 기본 비유와 배치는 일지가 맡고, 분석기에서는 데이터를 읽는 방식만 가져온다.

장면: 터미널과 편집기 옆에 띄워 두고 실패 원인을 좁혀 가는 개발자와 QA다. 낮에는 밝은 장부 용지, 밤에는 같은 괘선에 밝은 잉크를 쓴 어두운 용지를 쓴다. 판은 시스템 설정(`prefers-color-scheme`)을 따른다.

## Colors

- 용지와 괘선이 바탕이다. 영역은 상자가 아니라 괘선으로 나눈다. 큰 구획은 2px `ledger-ruling`, 줄 사이는 1px `ledger-ruling-faint`로 긋는다.
- 잉크는 넷이다. 검정은 본문, 청흑색은 링크와 선택과 기록계의 선, 초록은 성공, 황토색은 경고에 쓴다.
- 빨간 잉크는 잘못된 것에만 쓴다. 사고 기록, 실패한 테스트와 단계, 오류 로그, 다른 모든 노드와 어긋난 계기 값이 그 대상이다. 단순히 값이 바뀌었다거나 필터를 골랐다는 이유로는 쓰지 않는다.
- 수집 누락 구간은 `gap-fill` 음영으로 칠하고, 그 안에서는 값을 "—"로 적는다.
- 색은 `web/src/theme.css`의 토큰으로만 쓴다. 컴포넌트에 색 값을 직접 적으면 `design_tokens.test.mjs`가 막는다. 어두운 판의 값도 같은 파일의 `prefers-color-scheme: dark` 블록에 있다.

## Typography

- 장의 제목과 사고 기록은 sans 굵은체로 쓴다. 사고 기록 문장(`--text-display`, 30px)이 한 쪽에서 가장 크다.
- 칸 이름, 열 머리, 범례는 condensed 굵은체로 쓰고 자간을 조금 벌린다. 한국어가 섞이므로 대문자 변환은 쓰지 않는다. 엔진 표기(`node2`, `transfer-native`)는 쓰인 그대로 둔다.
- 시각, 수치, 로그 본문, 엔진 코드는 고정폭에 `tabular-nums`로 쓴다.
- 글꼴은 시스템에 있는 것을 쓴다. 웹 글꼴 파일은 아직 넣지 않았다.

## Layout

- 셸은 왼쪽 쪽번호 목차(01, 02 …)와 본문으로 나뉜다. 둘 사이에는 2px 괘선을 긋는다.
- 800px 아래에서는 목차가 본문 위로 올라와 한 줄에 세 쪽씩 두 줄로 놓인다. 옆으로 밀어야 보이는 쪽을 두지 않는다.
- 터치 화면(`pointer: coarse`)에서는 누를 수 있는 것을 모두 44px 이상으로 둔다. 버튼, 입력칸, 선택칸은 `--control-height`가 44px로 바뀌고, 접힘 제목과 글자 링크는 높이를 따로 늘린다. 체크박스는 그린 크기는 그대로 두고 감싼 칸을 키운다.
- 작업 상세는 테스트 실행 기록을 다 읽기 전에는 사고 기록에 "테스트 판정을 읽는 중"만 적고 단계 화면을 띄우지 않는다. 페이지 끝의 엔진 세션은 세션마다 기록 상세로 건너가고, 확인이 필요한 자원은 황토색 괘선 아래 경로를 한 줄씩 적는다. 사고 기록의 "보관 기록"은 기록 상세의 ID이고 엔진 세션 이름과 다르다.
- 작업 상세는 위에서부터 이 순서로 놓는다. 공용 화면 제목은 머리줄과 같은 말을 두 번 하므로 화면 읽기 프로그램에만 남긴다. 실행 중인 작업은 사고 기록의 걸린 시간 칸에 지난 시간을 1초마다 늘려 적는다. 머리줄, 사고 기록(시각 칸 | 문장 | 작업 정보), 기록계 띠(채널 | 측정 칸), 로그(왼쪽)와 계기 기록표(오른쪽), 테스트 기록, 단계별 당직 기록이다.
- 기록 상세는 작업 상세와 같은 순서를 따른다. 머리줄, 사고 기록, 기록계 띠와 로그·계기 기록표(`RunBench`), 테스트 기록, 단계 기록, 기록 정보 장부 순서다. 테스트 판정이 없는 실행은 실행 구간 전체를 띠로 읽는다.
- 읽던 증거는 주소로 남긴다. 작업 상세와 기록 상세의 URL은 고른 실행(`run`), 테스트(`test`), 로그 노드(`node`), 옮긴 커서 B의 시각(`at`)을 담아, 복사한 주소를 열면 같은 자리에서 시작한다. 다른 테스트를 고르거나 커서를 기본 위치로 돌리면 `at`을 지운다. 대시보드 최근 결과의 실행 기록 칸은 실행마다 기록 상세 주소로 건너간다.
- 대시보드는 일지의 표지다. 맨 위에 괘선으로 나눈 합계 칸, 그 아래 빨간 괘선으로 연 "주의가 필요한 작업" 장부, 실행 중 작업, 괘선 격자로 그린 노드 상태, 최근 결과 순서다. 노드 칸은 색으로 채우지 않는다.
- 모니터링의 작업과 노드는 장부 표다. 760px 아래에서는 공용 `table.stack` 규칙으로 줄마다 칸 이름과 값을 쌓는다.
- 900px 아래에서는 칸이 세로로 쌓인다. 띠의 시간축 눈금은 절반만 남긴다. 700px 아래에서는 단언 표가 칸 이름과 값을 쌓은 목록이 되고, 긴 숫자는 줄을 바꾸지 않고 옆으로 넘긴다.

## Elevation & Depth

그림자를 쓰지 않는다. 깊이는 괘선의 굵기와 용지 단계(`--bg`, `--surface`, `--surface-raised`)로 나타낸다. 고른 줄과 펼친 칸만 한 단계 밝은 용지에 놓인다.

## Shapes

모서리는 모두 직각이다(`--radius: 0`). 둥근 모양은 두 곳에만 쓴다. 하나는 사고 기록에서 탓이 확인된 노드 이름을 둘러친 원이고, 다른 하나는 이상한 계기 값을 둘러친 빨간 원이다. 상태 표시(StatusMark)는 같은 굵기의 선으로 그린 SVG다. 체크, 가위표, 점, 반원, 느낌표, 빈 원, 물음표가 있으며 유니코드 글자는 쓰지 않는다.

## Components

- **사고 기록 (CasualtyEntry)**: 시각 칸에는 실패 시각을, 끝난 작업이면 끝난 시각을 적는다. 문장에서는 단언이 걸린 노드를 적되, 기록계 띠의 기본 커서에서 그 노드의 계기 값이 다른 모든 노드와 어긋나거나 오류 로그가 있을 때만 원으로 두른다. 근거 줄에서는 기대값에 줄을 긋고 실제값과 차이를 빨갛게 적는다. 그 아래 띠와 같은 판정 문장을 한 줄 적는데, 노드에 이상이 없으면 기대값이나 시나리오를 먼저 보라고 덧붙인다. 띠를 읽기 전에는 판정도 원도 두지 않는다. 그 아래 실패 경로(단계, 테스트, 노드 로그)를 둔다. 상태는 기울인 도장으로 찍는다. 성공한 작업은 통과한 테스트 수를 문장으로, 앞의 세 테스트 이름과 걸린 시간을 근거 줄로 적고, 첫 실행을 펼쳐 둔 채 작업 정보를 접는다.
- **기록계 띠 (RecorderStrip)**: 채널마다 왼쪽에 이름과 요약을 둔다. 띠를 누르면 고른 커서가, Shift를 누른 채 누르면 다른 커서가 옮겨진다. 화살표는 1초씩 옮긴다. 측정 칸에 "A 옮기기 / B 옮기기 / 기본 위치로 / 이전 오류 / 다음 오류"가 있다. 띠에 초점이 있을 때 `]`와 `[`는 고른 커서를 이 노드의 다음 오류와 이전 오류로 옮긴다. 기본 위치는 A가 첫 오류 전의 첫 경고(없으면 구간 시작), B가 첫 오류(없으면 테스트 끝)다. 단계가 실패한 작업의 띠에는 작업 단계를 맨 위 채널로 그리고 실패한 단계에 이름을 붙인다. B 커서는 A처럼 잉크로 그리고 오류 위에 놓일 때만 빨갛게 한다. 커서 이름표는 축 눈금 아래 띠 맨 위의 자기 칸에 두고, 막대 이름은 막대 뒤(막대가 늦게 끝나면 앞)에 잉크로 적는다. 측정 칸은 맨 위에 값만으로 만든 판정 한 줄을 둔다. B 시점에 다른 모든 노드와 어긋나는 계기, 블록과 다른 노드 최고 블록, 구간의 오류 수를 적고 원인을 추정하지 않는다. 터치 화면에서는 키보드 안내 대신 터치 안내를 보인다.
- **계기 기록 (Readings)**: 표본마다 한 줄이다. 다른 모든 노드보다 뒤처진 블록, 더 적은 피어, 더 많은 대기 tx를 빨간 원으로 두른다. 측정 칸의 빨간 값도 같은 규칙을 따른다.
- **테스트 기록 (TestLedger)**: 실패와 막힘을 먼저 늘어놓은 장부 줄이다. 고른 줄 아래에 단언 표와 스텝 오류가 펼쳐진다(`data-testid="run-cause"`).
- **제자리 확인 (ConfirmAction)**: 되돌릴 수 없는 동작(작업 취소, 자격증명 폐기, 보관 결과 삭제)은 브라우저 대화상자로 묻지 않는다. 대상 바로 아래에 빨간 괘선 두 줄 사이로 무슨 일이 생기는지 적고, "…확정" 버튼과 "돌아가기"를 둔다.
- **단계 표시 (PhaseSteps)**: 대시보드와 모니터링의 작업 줄에서 단계를 눈금으로 그리고, 끝난 수와 지금 단계(실패하면 그 단계)를 글로 적는다.
- **단계 기록 (PhaseWatches)**: 당직 순서대로 한 줄씩 적는다. 실패한 단계만 펼쳐지고, 그 줄은 빨간 겹줄로 닫는다.
- **로그 레벨 칩**: 고른 레벨은 테두리를 잉크로 칠하고 밑줄을 굵게 긋는다. 칸을 색으로 채우지 않는다.

## Do's and Don'ts

- 영역은 괘선으로 나눈다. 둥근 카드를 쓰거나 카드 안에 카드를 넣지 않는다.
- 빨간 잉크는 잘못된 것에만 쓴다. 강조나 선택을 표시하려고 쓰지 않는다.
- 상태는 그린 표시와 한국어 낱말, 엔진 코드를 함께 보인다. 색 하나로만 알리지 않는다.
- 누락 구간을 0이나 직전 값으로 그리지 않는다.
- 그림자, 그라데이션, 형광 강조, 유니코드 아이콘을 쓰지 않는다.
