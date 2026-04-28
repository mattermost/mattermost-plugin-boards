# Pages 기능 작업 체크리스트

> **연관 문서**: [PAGES_PLAN.md](./PAGES_PLAN.md)
> **상태 범례**: ⬜ 미시작 · 🔄 진행 중 · ✅ 완료 · ⚠️ 부분/주의 · ❌ 실패/blocker

## 진행 요약

| 단계 | 진행 | 비고 |
|---|---|---|
| (가) 한국어 IME PoC | ✅ 완료 | G1~G5 모두 완료. 결정: **Tiptap 채택** (양쪽 검증 모두 통과). 상세: `docs/research/G2-poc-results.md` |
| (나) DB 마이그레이션 SQL 초안 | ✅ | 000041_add_pages.up/down.sql 작성. sqlite 시범 실행 통과 (5 테이블) |
| (다) Go/TS 모델 골격 | ✅ | Go: 4 신규 / 4 수정. TS: 6 신규 / 3 수정. Go build & TS tsc 모두 0 에러. D4/D13 Phase 1 으로 deferred |
| (라) Phase 1 본 작업 진입 | ✅ | Vertical slice 13/13 통과. e2e API 검증 한국어 round-trip OK. UI 사이드바 통합은 Phase 1 후속으로 분리 |

---

## 단계 (가) — 한국어 IME PoC

> **목적**: Tiptap (ProseMirror 기반) 가 한국어 composition event 를 안전하게 처리하는지 확인. 막히면 Lexical / 자체 구현 등 대안 평가.

| ID | 항목 | 상태 | 결과 / 노트 |
|---|---|---|---|
| G1 | 알려진 Tiptap+Korean IME 이슈 리서치 (웹 검색·이슈 트래커) | ⚠️ | **활성 미해결 버그 2건 발견.** Tiptap#5605 (한+Enter→마지막글자 소실), Tiptap#5250 (한+Yjs 협업 cursor→소실). 우리 조합 직격. ProseMirror 본가는 2026-04-07 archived. **상세**: `docs/research/G1-tiptap-korean-ime.md` |
| G1.1 | 대안으로 **Lexical (Meta)** 평가 추가 | ✅ | `@lexical/yjs` 공식, "Korean IME workaround" 가 changelog 에 명시됨. PoC 단계에서 Tiptap 와 양쪽 검증으로 확대 |
| G2 | PoC 격리 환경 셋업 (`/root/mattermost-plugin-boards/poc/editor-ko-ime/`) — Tiptap 과 Lexical 둘 다 검증 가능한 최소 Vite+React 앱 | ✅ | 파일 12개 작성. BroadcastChannel 기반 cross-tab Yjs sync. localStorage 결과 저장. npm install 완료 (163 패키지, 104MB). `npm run build` 성공 (TS 0 에러, 740KB 번들). 실행: `cd .../poc/editor-ko-ime && npm run dev` → http://localhost:5173 |
| G3 | 입력 시나리오 검증 (S1~S10) | ✅ | 사용자가 Tiptap 와 Lexical 양쪽에서 S1~S10 직접 입력 검증. **양쪽 모두 전 시나리오 통과**. 알려진 버그 (#5605 한+Enter, #5250 한+Yjs) 우리 환경에서 미재현 |
| G4 | PoC 결과 문서화 → `docs/research/G2-poc-results.md` | ✅ | 시나리오별 결과표 + 버전 정보 + 한계 포함 |
| G5 | 의사결정 | ✅ | **Tiptap 채택** — PAGES_PLAN.md 원안 그대로. 모니터링: ProseMirror archived → Phase 2-3 도중 결함 발생 시 Lexical 교체 재평가 |

---

## 단계 (나) — DB 마이그레이션 SQL 초안

> **목적**: Phase 1 에 필요한 모든 DB 변경을 마이그레이션 파일로 작성. PostgreSQL / MySQL / SQLite 모두 호환.

| ID | 항목 | 상태 | 결과 / 노트 |
|---|---|---|---|
| N1 | 마이그레이션 번호 결정 | ✅ | 000041 (본가 마지막 000040 다음) |
| N2 | `boards.layout` ALTER | ✅ | `addColumnIfNeeded` 패턴 적용, boards + boards_history |
| N3 | `page_content` 테이블 | ✅ | tiptap_json + yjs_state(BYTEA/MEDIUMBLOB/BLOB) + 메타 |
| N4 | `page_yjs_updates` 테이블 | ✅ | autoincrement (BIGSERIAL/AUTO_INCREMENT/AUTOINCREMENT) |
| N5 | `page_channels` 테이블 | ✅ | 다대다 PK (page_id, channel_id) |
| N6 | `page_acl` 테이블 | ✅ | 4-flag scheme (Boards 패턴) |
| N7 | `page_links` 테이블 | ✅ | 백링크용. anchor 컬럼 포함 |
| N8 | `page_revisions` 테이블 | ⏸ defer | Phase 3 진입 시 별도 마이그레이션으로 추가 |
| N9 | 멀티 DB 템플릿 처리 | ✅ | postgres/mysql/sqlite 분기 모두 박힘 |
| N10 | down 마이그레이션 | ✅ | 역순 DROP + dropColumnIfNeeded |
| N11 | 인덱스 | ✅ | page_yjs_updates(page_id,id), page_channels(channel_id), page_acl(user_id), page_links(dst_page_id) |
| N12 | 로컬 sqlite 시범 실행 | ✅ | CREATE TABLE 부분 통과. helper 런타임 미실행 |

---

## 단계 (다) — Go / TS 모델 골격

> **목적**: 빈 껍데기까지 잡아두기. CRUD 로직은 Phase 1 에서 채움.

### Go

| ID | 항목 | 상태 | 결과 / 노트 |
|---|---|---|---|
| D1 | `server/model/block.go` 에 `BlockTypePage = "page"` 상수 추가 | ✅ |  |
| D2 | `server/model/board.go` 에 `Board.Layout` 필드 + `BoardLayoutBoard / BoardLayoutDoc` 상수 | ✅ |  |
| D3 | `server/model/page.go` 신설 — Page / PagePatch / PageMember 구조체 | ✅ |  |
| D4 | `server/services/store` 인터페이스에 PageStore 메서드 시그니처 추가 | ⏸ defer | Phase 1 으로 이월 — Store interface 확장은 모든 implementation (sqlstore, mockstore 등) 에 stub 추가 부담. 지금은 sqlstore 의 concrete 메서드만 있음 |
| D5 | `server/services/store/sqlstore/page.go` 신설 — 빈 구현 (TODO) | ✅ |  |
| D6 | `server/app/pages.go` 신설 — PageService 골격 | ✅ |  |
| D7 | `server/api/pages.go` 신설 — REST 엔드포인트 골격 (`/api/v2/pages/...`) | ✅ |  |
| D8 | `server/services/permissions/mmpermissions/mmpermissions.go` 에 `HasPermissionToPage` 추가 (board 권한 + page_acl override) | ✅ |  |

### TS (webapp)

| ID | 항목 | 상태 | 결과 / 노트 |
|---|---|---|---|
| D9 | `webapp/src/blocks/page.ts` — Page / PageContent 타입 | ✅ |  |
| D10 | `webapp/src/pages/` 디렉토리 — 컴포넌트 골격 (DocSidebar, PageView, PageEditor, PageBreadcrumb) | ✅ |  |
| D11 | `webapp/src/store/pages.ts` — Redux slice (pagesAdapter) | ✅ |  |
| D12 | `webapp/src/octoClient.ts` 에 페이지 API 클라이언트 메서드 추가 | ✅ |  |
| D13 | `webapp/src/router.tsx` 또는 동등 위치에 `/page/:slug` 라우트 추가 | ⏸ defer | Phase 1 으로 이월 — 컴포넌트가 빈 stub 이라 dead route 만 추가됨. PageView 가 실제 동작 시점에 등록 |
| D14 | i18n 키 placeholder 추가 (en.json / ko.json) | ✅ |  |

---

## 단계 (라) — Phase 1 본 작업 진입

> 위 (가)~(다) 가 모두 ✅ 또는 ⚠️(허용 가능) 일 때만 시작.

| ID | 항목 | 상태 | 비고 |
|---|---|---|---|
| P1.0 | (가)~(다) 결과 종합 → Phase 1 작업 분해 | ✅ | Vertical Slice 13개 단위 (VS1~VS13) 로 분해 |

### Phase 1 — Vertical Slice 13단계

목표: "doc-board 생성 → 페이지 1장 생성 → 본문 입력 → 저장 → 새로고침 시 영속" end-to-end

| ID | 항목 | 상태 | 결과/노트 |
|---|---|---|---|
| VS1 | Store interface 에 Page 메서드 시그니처 추가 | ✅ | 5 메서드 추가 + mockstore 에 hand-written mock 추가. Go build OK |
| VS2 | sqlstore 미니멀 CRUD 5종 (CreatePage / GetPage / GetChildPages / GetPageContent / UpsertPageContent) | ✅ | squirrel 패턴, JSON fields encode/decode, UPDATE → INSERT upsert. Go build OK |
| VS3 | Board.Layout 필드 wiring (struct + boardFields + scan + INSERT/UPDATE) | ✅ | model/board.go (Board, BoardPatch, Patch, IsValid). sqlstore/board.go 7개 site (boardFields, boardHistoryFields, boardsFromRows scan, insertBoard map+UPDATE, deleteBoardAndChildren map, duplicateBoard columns+values). 빈 값 시 'board' 디폴트 자동 셋. Go build OK |
| VS4 | app/pages.go 본 구현 (ID 생성, 권한, WS broadcast) | ✅ | CreatePage / GetPage / GetChildPages / GetPagesForBoard / GetPageContent / SavePageContent. doc-board 검증, parent 검증, WS broadcastPageChange. 기타 메서드는 stub |
| VS5 | api/pages.go 핸들러 본 구현 + 라우트 등록 | ✅ | 6개 vertical slice 핸들러 본 구현 + 7개 Phase 2+ stub. api.go 에서 registerPagesRoutes 호출. 권한 검사 완료 |
| VS6 | webapp 에 Tiptap 의존 추가 | ✅ | @tiptap/react,pm,starter-kit + extension-collaboration,collaboration-cursor + yjs + y-protocols. legacy-peer-deps 옵션 사용. tsc OK |
| VS7 | Redux thunks + slice 등록 | ✅ | fetchPagesForBoard / fetchPage / fetchPageContent / createPage / savePageContent thunk + selectors (getPage, getChildPageIds, getChildPages, getAncestorTrail, getPageContent). store/index.ts 에 pagesReducer 등록 |
| VS8 | DocSidebar 트리 렌더 | ✅ | 재귀 PageTreeNode + chevron 토글 + add 버튼. Phase 2 에 DnD 추가 예정 |
| VS9 | PageEditor — Tiptap 인스턴스 + 디바운스 저장 | ✅ | useEditor + StarterKit + 800ms 디바운스 savePageContent + 저장 상태 표시. Yjs 통합은 Phase 2 |
| VS10 | PageView 통합 (제목 + 에디터 + breadcrumb) | ✅ | breadcrumb + h1 + PageEditor. 제목 inline 편집은 Phase 1 후속 |
| VS11 | 라우터 등록 (DocPage 컴포넌트, `/team/:teamId/doc/:boardId/:pageId?`) | ✅ | router.tsx 에 추가. URL 직접 진입 가능 |
| VS12 | doc-board 생성 진입점 | ⚠️ | curl + URL 직접 입력 방식. UI 진입점 (사이드바 통합) 은 Phase 1 후속 작업으로 별도 분리. 상세: `docs/research/VS12-entry-points.md` |
| VS13 | 빌드 + 배포 + 수동 검증 한 사이클 | ✅ | 빌드/업로드/활성화 OK. e2e API 검증 통과 (한국어/이모지 round-trip 정확 보존). 발견한 3건 버그 모두 수정 (channel_id NOT NULL, JSONB string 캐스팅, RawMessage 응답) |

---

## 변경 로그

| 일시 | 항목 | 변경 |
|---|---|---|
| 2026-04-26 | 초기 작성 | 단계 (가)~(라) 골격 생성 |
| 2026-04-26 | G1 완료 (⚠️) | Tiptap 미해결 버그 2건 발견. PoC 범위 Lexical 추가로 확대 (G1.1, G2 갱신) |
| 2026-04-27 | G2 완료 (✅) | PoC Vite+React 앱 작성. Tiptap/Lexical/Yjs/BroadcastChannel/시나리오 체크리스트 포함. npm install 완료 |
| 2026-04-27 | G3~G5 완료 (✅) | 사용자가 양쪽 에디터 검증, 둘 다 전 시나리오 통과. **Tiptap 채택 결정**. 단계 (가) 종료, (나) 진입 가능 |
| 2026-04-27 | (나) 완료 (✅) | 000041_add_pages.up/down.sql 작성. sqlite 시범 실행 통과. 단계 (다) 진입 가능 |
| 2026-04-27 | (다) 완료 (✅) | Go 4 신규 + 4 수정, TS 6 신규 + 3 수정. Go build & TS tsc 0 에러. D4(Store interface), D13(라우터) 은 Phase 1 으로 이월 |
| 2026-04-27 | VS1+VS2 완료 (✅) | Store interface 확장 + 5개 sqlstore CRUD 본 구현 (CreatePage/GetPage/GetChildPages/GetPageContent/UpsertPageContent). Go build OK |
| 2026-04-27 | VS3 완료 (✅) | Board.Layout 필드 wiring 9곳 (model 4 + sqlstore 5 sites). 빌드 OK |
| 2026-04-27 | VS4 완료 (✅) | app/pages.go: CreatePage / GetPage / GetChildPages / GetPagesForBoard / GetPageContent / SavePageContent + broadcastPageChange |
| 2026-04-27 | VS5 완료 (✅) | api/pages.go 6 vertical slice 핸들러 + 7 stub. registerPagesRoutes 등록 |
| 2026-04-27 | VS6~VS10 완료 (✅) | Tiptap+Yjs deps + Redux thunks/slice 등록 + DocSidebar/PageEditor/PageView/PageBreadcrumb 본 구현 |
| 2026-04-27 | VS11~VS13 완료 (✅) | DocPage 컴포넌트 + 라우트 + e2e 검증 통과. 한국어 round-trip OK. **Phase 1 Vertical Slice 13/13 완료** |
| 2026-04-27 | **모델 변경: X→Y** | 사용자 결정 — 페이지를 team-scoped 1급 객체로 격상. 양방향 board↔page 참조. 기존 doc-board 모델은 deprecated. v0.9.2 업로드 보류, Y9 끝까지 후 재배포 |
| 2026-04-27 | Y1 완료 (✅) | 마이그레이션 000042: pages 테이블 + board_page_refs + page_board_refs + 더티 데이터 정리. sqlite 시범 통과 |
| 2026-04-27 | Y2~Y9 완료 (✅) | model+sqlstore+app+api+TS 전체 재작성. v0.9.2 빌드+업로드+활성화. **e2e API 검증 통과**: team-scoped 페이지 CRUD + 양방향 board↔page cross-ref 모두 동작. 한국어 round-trip 정확 |
