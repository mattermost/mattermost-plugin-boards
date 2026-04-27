# G1 — Tiptap + Korean IME 알려진 이슈 리서치

> **연관**: [PAGES_CHECKLIST.md](../PAGES_CHECKLIST.md) 의 G1
> **결론 (요약)**: Tiptap + Yjs + 한국어 조합에서 **현재 미해결 활성 버그 2건** 확인. PoC 결과가 아주 중요해짐. 대안으로 **Lexical** 검토 필요.

## 발견된 미해결 버그

### Bug #1 — Korean + Enter 키 = 마지막 글자 소실

- **이슈**: [ueberdosis/tiptap#5605](https://github.com/ueberdosis/tiptap/issues/5605)
- **재현**: 한국어 입력 → Enter → 마지막 글자가 사라짐
- **브라우저**: Chrome
- **버전**: 최신 (보고 시점 기준 2024-09)
- **상태**: **OPEN, fix 없음, workaround 미문서화**
- **우리 영향**: **매우 큼** — 매뉴얼·온보딩 작성에서 Enter 는 가장 흔한 입력. 한국어 사용자에게 즉시 발견될 결함

### Bug #2 — Korean + Yjs collaborator cursor = 텍스트 소실

- **이슈**: [ueberdosis/tiptap#5250](https://github.com/ueberdosis/tiptap/issues/5250)
- **재현**: 협업 cursor 활성 상태에서 한국어 입력 → 다른 위치 클릭 → 입력했던 한글이 사라짐
- **브라우저**: Chrome
- **버전**: Tiptap 2.2.2 (2024-06 보고)
- **상태**: **OPEN, fix 없음**
- **우리 영향**: **치명적** — 우리 계획의 핵심 조합 (Tiptap + Yjs + Korean) 이 정확히 이 케이스. 영어 입력에서는 발생 안 함

## 우리 시나리오와의 매핑

우리 plan: **Pages 에디터 = Tiptap + Yjs + 한국어 1차 사용자**

| 시나리오 | Bug 영향 |
|---|---|
| 한국어로 매뉴얼 본문 작성 | Bug #1 직격 (Enter 매번) |
| 두 명이 동시에 한국어 페이지 편집 | Bug #2 직격 |
| 한국어 페이지 작성 후 다른 페이지로 이동 | Bug #2 가능성 |

## 추가 정황

- **ProseMirror repo 가 2026-04-07 archive 처리 됨** ([저장소](https://github.com/ProseMirror/prosemirror)) — Tiptap 의 기반 라이브러리가 read-only 상태. 새 IME 버그 fix 가 ProseMirror 코어에서 와야 한다면 어렵게 됨
- Tiptap 팀의 최근 IME 관련 fix:
  - Vue 3 mark view 의 DOM 파괴 방지 (composition 도중)
  - TOC update 를 IME composition 중 skip
  - 그러나 **핵심 두 버그는 미해결**

## 대안 — Lexical (Meta) 평가

[Facebook Lexical](https://lexical.dev) 은 다른 접근:

| 비교 | Tiptap | Lexical |
|---|---|---|
| 기반 | ProseMirror (archived) | 자체 reconciler |
| Yjs 통합 | `y-prosemirror` | `@lexical/yjs` (공식) |
| Korean IME | Bug #5605, #5250 미해결 | **iOS Korean IME workaround 명시 적용**됨 |
| 문서/예제 | 풍부 | 풍부 |
| 한국 사용 사례 | (확인 필요) | Notion·Workplace 가 Lexical 기반 (확인 필요) |
| 라이선스 | MIT | MIT |
| 노드 시스템 | 풍부 | 풍부 (table/code/list 기본) |
| 팀 활동 | 활발 | 활발 (Meta 1st-party) |

Lexical 이 **Korean IME 문제를 명시적으로 다룬다**는 점이 결정적 차별 요소.

## PoC 갱신 필요 사항

원래 G2 (PoC 환경 셋업) 가 **단순 Tiptap 검증** 이었는데, 이제 **Tiptap + Lexical 양쪽** 검증으로 확대해야 함:

- Tiptap PoC — Bug #5605, #5250 재현 시도 (최신 버전이라도 남아있는지)
- Lexical PoC — 같은 시나리오에서 안 터지는지

검증 시나리오 (둘 다 동일하게):
1. 한국어 단독 입력 (한 단락)
2. 한국어 + Enter (Bug #1 재현 표적)
3. 영한 혼용 입력
4. 멘션 시작 (`@`) 도중 한국어 IME 진입
5. 리스트/체크박스 항목 안에서 한국어 IME
6. **Yjs awareness 활성 상태에서** 한국어 입력 (Bug #2 재현 표적)
7. 두 탭 띄워 동시 편집 + 한국어 입력
8. composition 도중 Backspace, Arrow 키
9. 빠른 연속 입력

## 의사결정 분기

PoC 결과에 따라:

| PoC 결과 | 결정 |
|---|---|
| Tiptap 양호 / 두 버그 재현 안 됨 | **Tiptap 진행** (계획 그대로) |
| Tiptap 일부 결함 / Lexical 양호 | **Lexical 로 전환** — 계획에서 `@tiptap/*` → `lexical` + `@lexical/yjs` 로 라이브러리 교체. UI 모델 약간 다르지만 노드 셋팅 비슷 |
| 둘 다 결함 | 옵션 분기: (a) 결함 받아들이고 진행 / (b) ProseMirror 코어 직접 패치 (archived 라 위험) / (c) 자체 구현 (대규모 작업, 거의 비추천) |
| 둘 다 양호 | **Lexical 권장** — Korean IME 를 의식적으로 다루는 팀 + 활발한 팀 |

## 영향 — 견적과 일정

PAGES_PLAN.md 의 견적이 바뀌지는 않음 (Tiptap 이든 Lexical 이든 비슷한 추상화 수준). 그러나:

- **PoC 작업이 1-2일 → 2-3일**로 늘어남 (양쪽 검증)
- **막힐 경우 분기 결정**에 추가 1-2일 (ProseMirror 패치 검토 등)
- **만약 Lexical 채택**: PAGES_PLAN.md 부록 B 의 의존 라이브러리 표 갱신 필요

## 참고 링크

- [Tiptap Issue #5605 — Korean + Enter](https://github.com/ueberdosis/tiptap/issues/5605)
- [Tiptap Issue #5250 — Korean + Collaborator cursor](https://github.com/ueberdosis/tiptap/issues/5250)
- [ProseMirror — Korean IME Chrome 128](https://github.com/ProseMirror/prosemirror/issues/1484)
- [Tiptap CHANGELOG (IME fixes)](https://github.com/ueberdosis/tiptap/blob/develop/CHANGELOG.md)
- [Lexical 공식 사이트](https://lexical.dev)
- [@lexical/yjs npm](https://www.npmjs.com/package/@lexical/yjs)
- [Lexical Yjs 가이드](https://lexical.dev/docs/packages/lexical-yjs)
