# G2 PoC 결과 + G5 의사결정

> **연관**: [PAGES_CHECKLIST.md](../PAGES_CHECKLIST.md), [G1 리서치](./G1-tiptap-korean-ime.md)
> **테스트 일시**: 2026-04-27
> **PoC 위치**: `/root/mattermost-plugin-boards/poc/editor-ko-ime/`

## 검증된 시나리오

PoC 의 S1 ~ S10 시나리오를 Tiptap 과 Lexical 양쪽에서 직접 입력 검증.

| ID | 시나리오 | Tiptap | Lexical |
|---|---|---|---|
| S1 | 한글 단독 입력 | O | O |
| S2 | 한글 + Enter (Bug #5605 표적) | **O** | O |
| S3 | 영한 혼용 | O | O |
| S4 | 리스트 항목 IME | O | O |
| S5 | 헤딩 안 IME | O | O |
| S6 | 코드 블록 안 IME | O | O |
| S7 | Yjs 협업 도중 한국어 (Bug #5250 표적) | **O** | O |
| S8 | 두 탭 동시 입력 | O | O |
| S9 | Composition 도중 Backspace / Arrow | O | O |
| S10 | 빠른 연속 입력 | O | O |

**결과 요약**: 두 에디터 모두 모든 시나리오 통과. 알려진 Tiptap 버그 (#5605, #5250) 우리 환경에서 재현 안 됨.

## 사용된 버전

- Tiptap: 2.10.x (`@tiptap/react`, `@tiptap/pm`, `@tiptap/starter-kit`, `@tiptap/extension-collaboration`, `@tiptap/extension-collaboration-cursor`)
- Lexical: 0.21.x (`lexical`, `@lexical/react`, `@lexical/yjs`, `@lexical/rich-text`, `@lexical/list`, `@lexical/utils`)
- Yjs: 13.6.20
- React: 18.3.1
- Vite: 5.4.x

## 검증 환경

- Node 24.14.1 / npm 11.11.0
- Yjs sync: BroadcastChannel 기반 cross-tab (서버 없이)
- 두 탭 띄워 직접 입력으로 검증

## G5 의사결정 — Tiptap 채택

### 결정

PAGES_PLAN.md 의 원안대로 **Tiptap + Yjs 진행**. Lexical 은 백업 옵션으로 유지.

### 근거

1. 현장 검증에서 알려진 버그 재현 안 됨
2. PAGES_PLAN.md 와 의존 라이브러리 표 변경 0
3. Tiptap StarterKit 의 노드 풍부 — Phase 1 작업 시간 단축
4. y-prosemirror 통합이 검증된 조합

### 모니터링 항목

ProseMirror 본가 archived (2026-04-07) — Tiptap 의 기반이 유지보수에서 멀어진 상태. Phase 2-3 진행 중 새로운 IME / Yjs 결함 발생 시 Lexical 교체 옵션 재평가.

## 미테스트 / 한계

- macOS 환경에서 한국어 IME 검증 안 됨 (테스트 환경 한정)
- 모바일 환경 미검증 (Phase 4 모바일 정책 시 별도 검증 필요)
- 1시간+ 장시간 입력 미검증
- 1000자 이상 대용량 문서 미검증

위 항목들은 Phase 1 진행 중 발견되면 별도 ticket 으로 처리.
