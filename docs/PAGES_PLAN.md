# Mattermost Boards — Pages 기능 계획서 (v1)

> **상태**: working draft v1 (2026-04-26)
> **목적**: Slack Canvas 대체 기능을 Mattermost Boards 플러그인 안에 추가
> **주 사용처**: 매뉴얼 작성, 온보딩 가이드

---

## 1. 개요

### 1.1 배경

- Mattermost 에 Slack Canvas / Notion / Outline 급의 협업 문서 기능이 없음
- Boards 의 카드 description 만으로는 매뉴얼·온보딩 운영이 어려움 (위계 부족, 책 분량 어려움)
- 외부 도구 (Outline, BookStack 등) 도입은 SSO 가 있어도 UI 셸 이질감 큼

### 1.2 목표

- Mattermost Boards 플러그인 **안에서** 동작하는 협업 문서 기능
- 매뉴얼/온보딩 시나리오에서 Slack Canvas 와 동등하거나 그 이상의 가치
- 사용자 학습 부담 최소화 — Boards 의 권한·공유·UI 패턴 재사용

### 1.3 비-목표 (Out of Scope)

- Slack workflow / Block Kit / Salesforce 임베드
- Boards 카드 자체를 Tiptap+Yjs 로 마이그레이션 (검증 후 Phase 5 옵션)
- 외부 도구로의 양방향 동기화 (Confluence 등)
- 화상회의(Calls) 안 페이지 공유

---

## 2. 아키텍처 결정 — 옵션 B 채택

### 2.1 후보 비교 (결정된 옵션 강조)

| | A. Boards 블록 시스템 재사용 | **B. Pages 만 Tiptap+Yjs, Boards 카드는 그대로** | C. Boards 카드부터 Tiptap+Yjs 마이그레이션 |
|---|---|---|---|
| 견적 | 4주 | **7-8주** | 12-15주 |
| 이질감 | 거의 없음 | UI chrome 통일로 최소화 | 마이그레이션 도중 더 큼 |
| 실시간 협업 | 블록 단위 last-write-wins | **Yjs CRDT** | Yjs CRDT |
| 본가 fork 분기 | 작음 | 중간 | 큼 (영구 분리) |
| 데이터 위험 | 없음 | 없음 (신규 테이블만) | 큼 (기존 데이터 변환) |

### 2.2 옵션 B 의 핵심 원칙

1. **Pages 는 Boards 플러그인 내부의 새 도메인** — 별도 플러그인 아님
2. **재사용** — 권한 검사 / WebSocket / 사이드바 컴포넌트 / 인증 / 라우팅 / 검색은 Boards 인프라 그대로
3. **신규** — Tiptap 에디터 + Yjs CRDT 협업 + 페이지 트리는 신설
4. **시각 일관성** — Boards 의 디자인 토큰 (색상, 폰트, 버튼 스타일) 그대로 입혀 chrome 이질감 제거

### 2.3 Notion 식 분리 정당화

- 카드 = 짧은 정형 레코드 (이슈, OKR, 콘텐츠 아이템) → 블록 에디터로 충분
- 페이지 = 긴 비정형 문서 (매뉴얼, 온보딩) → Tiptap+Yjs 가 적합
- Notion 도 데이터베이스 row 셀 편집과 페이지 편집은 다른 모드. **합리적 분리**

---

## 3. 데이터 모델

### 3.1 스키마 diff (현재 Boards 대비)

```sql
-- boards 테이블에 컬럼 추가
ALTER TABLE {{.prefix}}boards 
  ADD COLUMN layout VARCHAR(16) DEFAULT 'board';
-- 'board' = 기존 카드 보드
-- 'doc'   = 페이지 워크스페이스 (매뉴얼 한 권)

-- blocks.type 에 'page' 추가 (스키마 변경 없음, 타입 enum 만 확장)
-- blocks.parent_id 가 페이지 트리 표현
--   parent_id = boards.id  → 최상위 페이지
--   parent_id = blocks.id (type='page')  → 하위 페이지
```

### 3.2 Pages 전용 신규 테이블

```sql
-- 페이지 콘텐츠 (Tiptap 문서 + Yjs 상태)
CREATE TABLE {{.prefix}}page_content (
    page_id VARCHAR(36) PRIMARY KEY,           -- blocks.id 참조
    tiptap_json JSON,                          -- Tiptap 문서 (JSON 직렬화)
    yjs_state BYTEA,                           -- Yjs 압축 스냅샷 (binary)
    yjs_updates_count INT DEFAULT 0,           -- 마지막 스냅샷 이후 update 누적 수
    last_snapshot_at BIGINT,
    update_at BIGINT
);

-- Yjs update 로그 (스냅샷 사이의 증분)
CREATE TABLE {{.prefix}}page_yjs_updates (
    id BIGSERIAL PRIMARY KEY,
    page_id VARCHAR(36) NOT NULL,
    update_blob BYTEA NOT NULL,
    client_id VARCHAR(36),
    create_at BIGINT
);
CREATE INDEX ON {{.prefix}}page_yjs_updates (page_id, id);

-- 페이지 → 채널 다대다 링크
CREATE TABLE {{.prefix}}page_channels (
    page_id VARCHAR(36) NOT NULL,
    channel_id VARCHAR(36) NOT NULL,
    pinned_by VARCHAR(36),
    pinned_at BIGINT,
    PRIMARY KEY (page_id, channel_id)
);

-- 페이지별 ACL override (Phase 2)
CREATE TABLE {{.prefix}}page_acl (
    page_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    scheme_admin BOOLEAN,
    scheme_editor BOOLEAN,
    scheme_commenter BOOLEAN,
    scheme_viewer BOOLEAN,
    PRIMARY KEY (page_id, user_id)
);

-- 백링크 인덱스 (Phase 2)
CREATE TABLE {{.prefix}}page_links (
    src_page_id VARCHAR(36) NOT NULL,
    dst_page_id VARCHAR(36) NOT NULL,
    anchor VARCHAR(255),
    PRIMARY KEY (src_page_id, dst_page_id, anchor)
);

-- 페이지 명시 버전 (Phase 3)
CREATE TABLE {{.prefix}}page_revisions (
    id VARCHAR(36) PRIMARY KEY,
    page_id VARCHAR(36) NOT NULL,
    snapshot_blob BYTEA NOT NULL,
    label VARCHAR(255),
    author_id VARCHAR(36),
    create_at BIGINT
);
```

### 3.3 데이터 모델 시각화

```
boards (layout='doc')                ← "운영 매뉴얼" 책 1권
   ├── page block (parent_id=board)  ← "1장. 시작하기" (최상위)
   │     ├── page_content row
   │     └── page block (parent_id=above page)  ← "1.1 로그인" (하위)
   │           └── page_content row
   ├── page block "2장. 배포 절차"
   └── page block "3장. 모니터링"
```

### 3.4 Boards 카드와의 격리

- Pages 의 콘텐츠는 **`page_content`** 에 있음 (Tiptap JSON + Yjs 바이너리)
- Boards 카드의 콘텐츠 블록 (text/image/checkbox 등) 은 **그대로 `blocks` 에**
- 두 시스템이 같은 `blocks` 테이블의 type 만 다른 row 로 공존. 충돌 없음

---

## 4. 권한 모델

### 4.1 Boards 의 5축 그대로 활용

| 축 | 활용 |
|---|---|
| Board Type (Open/Private) | doc-board 도 동일 — Open 시 팀 멤버 자동 viewer |
| Channel binding | doc-board.channel_id 로 같은 메커니즘 — 채널 멤버 자동 editor (synthetic) |
| board_members ACL | doc-board 멤버 = 책 1권 권한자 |
| Role (Admin/Editor/Commenter/Viewer) | 그대로 |
| MinimumRole | 그대로 |
| Sharing token | 그대로 (페이지 공개 공유) |
| Team Admin elevation | 그대로 |

### 4.2 페이지 권한 — Notion 식 상속 (Phase 2)

```
페이지 P 의 effective permission =

  1. P 가 속한 doc-board 의 board_members 에서 시작
        ├── synthetic (채널 바인딩, Open+Template)
        ├── minimum_role 적용
        └── Team admin 격상

  2. P 의 page_acl override 가 있나?
        ├── 없음 → 1번 결과 그대로
        └── 있음 → 트리 위로 거슬러 올라가 가장 가까운 page_acl 적용
```

### 4.3 Phase 1 의 단순화

Phase 1 은 `page_acl` 없이 출발 — doc-board 의 board_members 만으로 책 1권 단일 정책. 사용해본 뒤 **"특정 절만 권한 다르게"** 시나리오가 자주 나오면 Phase 2 에 추가.

---

## 5. UI / UX 설계

### 5.1 사이드바

- 기존 Boards 사이드바에 **doc-board 는 트리 모드로 렌더**
- card-board 와 같은 컴포넌트, 표시 방식만 분기 (`<DocSidebar>` 컴포넌트)
- 페이지 항목에 chevron 으로 접기/펼치기
- 드래그앤드롭으로 페이지 이동/순서 변경
- 가상화 (1000+ 페이지 대비)

### 5.2 페이지 뷰

- 라우트: `/team/{teamId}/{boardId}/page/{pageSlug-shortid}`
- 컴포넌트 구조:
  ```
  <PageView>
    <PageBreadcrumb>            ← 부모 트레일
    <PageHeader>                ← 제목, 아이콘, 권한 버튼, 더보기
    <PageEditor>                ← Tiptap 인스턴스
    <PageComments>              ← Phase 2
  ```

### 5.3 권한 다이얼로그

- Boards 의 share dialog 컴포넌트 재사용 + 페이지 전용 항목 추가:
  - 부모 ACL 표시 ("부모로부터 상속")
  - "이 페이지부터 권한 다르게" 토글 (Phase 2)
  - 채널 링크 추가/제거 (페이지를 어느 채널에 핀)

### 5.4 삭제 컨펌 다이얼로그

```
┌─────────────────────────────────────────┐
│ "1장. 시작하기" 페이지를 삭제합니다       │
│                                          │
│ 이 페이지에 3개의 하위 페이지가 있습니다 │
│                                          │
│ ○ 하위 페이지도 모두 삭제                │
│ ● 하위 페이지는 부모로 이동              │
│                                          │
│           [ 취소 ]  [ 삭제 ]             │
└─────────────────────────────────────────┘
```

자식 없으면 단순 컨펌. 부모 승격이 깊이(10) 위반 시 경고 후 진행.

### 5.5 채널 통합

- 채널 헤더 우측에 "Pages" 버튼 — 클릭 시 그 채널에 핀된 페이지 목록 (page_channels)
- 페이지 안 더보기 메뉴 → "Pin to channel" → MM 채널 검색기

---

## 6. URL / 라우팅

### 6.1 형식

```
/team/{teamId}/{boardId}/page/{slug}-{shortid}

예시:
  /team/abc/def/page/환영-가이드-a3f9b2c1
  /team/abc/def/page/1-1-로그인-절차-b7e2d4a9
```

### 6.2 한국어 처리

- 한국어 그대로 IRI (RFC 3987) 형태로 사용
- 모든 모던 브라우저 지원, HTTP 레이어에서 percent-encode 자동
- 슬러그 길이: **최대 50자**, 초과 시 잘림
- shortid: **8자 영숫자**, 페이지 생성 시 nanoid 등으로 생성, 변경 불가

### 6.3 제목 변경 시

- 새 슬러그로 URL 갱신
- 옛 URL 도 shortid 로 매칭되면 **301 리다이렉트** → 새 URL
- 북마크/외부 링크 보호

### 6.4 검증된 정제 규칙

- 영숫자 / 한글 / 하이픈만 허용
- 공백 → 하이픈
- 연속 하이픈 → 하나로 압축
- 선두/말미 하이픈 제거
- 빈 슬러그 → `/page/-{shortid}` (선두 하이픈은 정제로 사라짐)
- emoji 제거 (페이지 이름엔 유지, URL 슬러그에서만 제거)

---

## 7. 실시간 공동 편집 — Yjs 패시브 릴레이

### 7.1 아키텍처

```
브라우저 A          브라우저 B          MM 서버 (focalboard 플러그인)
  │                   │                       │
  │  Y.Doc            │  Y.Doc                │  page_content (스냅샷)
  │                   │                       │  page_yjs_updates (증분)
  │                   │                       │
  ├─ 편집 ─→ update ──┼─→ MM WebSocket ───────┤
  │                   │                       │
  │                   ├← fanout to others  ───┤
  │                                            │
  └─ N분 또는 N updates 후 ─ snapshot ─────────→ DB 압축
```

- 클라가 Y.Doc 로컬 보유 → 편집 시 binary update 생성
- update 를 MM WebSocket plugin event 로 송신
- 서버는 **CRDT 계산 안 함** — fanout + 영속화만 (Go Yjs 미사용)
- 신규 접속자: 서버에서 마지막 스냅샷 + 그 이후 updates → Y.Doc 복원

### 7.2 압축 정책

- **5분마다 또는 100 updates 누적** 마다 트리거
- `page_yjs_updates` 의 update 들을 합쳐 새 `yjs_state` 스냅샷 생성
- 스냅샷 후 옛 update row 삭제 (DB 비대화 방지)

### 7.3 Awareness (presence)

- 누가 페이지를 보고 있는지 / 커서 위치 표시
- 클라 disconnect 후 30초 timeout 으로 ghost 정리

### 7.4 Phase 1 의 단순화 옵션

검증 단계로 **Phase 1 은 단일 편집자 모드 + 낙관적 락**으로 시작 가능. Yjs 레이어는 Phase 2 에 추가 — 데이터 모델은 동일하므로 비파괴적 확장.

---

## 8. 보안

| 항목 | 정책 |
|---|---|
| XSS | Tiptap 스키마 화이트리스트 + DOMPurify 추가 sanitization |
| 모든 WebSocket 메시지 권한 재검사 | 매번 (권한 변경 즉시 반영) |
| 파일 업로드 크기 | MM `FileSettings.MaxFileSize` 따름 |
| 게스트 사용자 | Boards 와 동일 — synthetic 멤버십 미부여, 명시 ACL 만 인정 |
| 한국어 IME 검증 | 본 작업 전 1-2일 PoC 필수 (Tiptap composition event 알려진 이슈) |

---

## 9. Export / Import

| 기능 | Phase | 비고 |
|---|---|---|
| Markdown export (페이지 1장) | **Phase 1** | Tiptap → MD 컨버터, 백업 가치 큼 |
| Markdown import (페이지 1장) | **Phase 1** | export 와 짝, 작업량 작음 |
| PDF export (브라우저 print) | **Phase 1** | print CSS 조정만 |
| PDF export (서버사이드, 폴리시드) | Phase 3 | Chromium headless 또는 wkhtmltopdf |
| 책 단위 묶음 export (Markdown / PDF) | Phase 3 | 트리 순회 + 합성 |
| Confluence/Notion import | 안 함 | 외부 도구 우회 가능 |

---

## 10. Phase 분할

### Phase 1 — 핵심 기능 (4-5주)

- DB 마이그레이션: `boards.layout` 컬럼, `page_content` / `page_yjs_updates` / `page_channels` 테이블
- Go: PageService, page CRUD API, 권한은 board_members 그대로
- TS: DocSidebar 트리 컴포넌트, PageView, Tiptap 에디터 셋업
- Tiptap 노드: heading H1-H3, paragraph, bold/italic/strikethrough/code, bulleted/ordered/checklist, table, code block, blockquote, callout, divider, image, link, mention
- 한국어 IME PoC 선행 (1-2일)
- Markdown export/import
- 단일 편집자 + 낙관적 락 (Yjs 미적용)

**Phase 1 완료 기준**: 사용자가 매뉴얼·온보딩 책 한 권을 작성/편집/공유 가능. 동시 편집은 한 명만.

### Phase 2 — 협업 강화 (1-2주)

- Yjs 통합 (page_yjs_state 사용 시작)
- 압축 정책 동작 (5분/100 updates)
- Awareness (presence cursors)
- page_acl override 테이블 + UI
- 채널 다대다 링크 UI
- 섹션 단위 코멘트
- 백링크 인덱스 + UI

### Phase 3 — 폴리시 (1-2주)

- 검색 강화 (권한 필터링, 워크스페이스/책 단위 모드)
- 페이지 멘션 알림 (MM 알림 후크)
- 책 단위 묶음 export
- 템플릿 갤러리 5종 (회의록, 프로젝트 브리프, 신입 온보딩, OKR, FAQ)
- 페이지 unfurl (MM 메시지에서 페이지 링크 → 카드 미리보기)
- 옛 URL 301 리다이렉트
- 휴지통 / 30일 복원

### Phase 4 — 운영 보강 (옵션, 1주)

- 모바일 읽기 + 텍스트 편집 (Slack Canvas 식 — 객체 추가 불가)
- 접근성 (키보드 네비, 스크린리더)
- 페이지 구독 알림
- 코멘트 resolve 상태

### Phase 5 — 미래 (옵션)

- Boards 카드 description 도 Tiptap 으로 점진 마이그레이션 (원할 때)
- AI 통합 (mattermost-ai 플러그인 연동 — "이 회의 정리해서 페이지 만들기")

---

## 11. 결정된 사항 — 전체 목록

### 거시 결정

| ID | 결정 | 출처 |
|---|---|---|
| M1 | 옵션 B 채택 (Pages 만 Tiptap+Yjs, Boards 카드 그대로) | 4/26 |
| M2 | 페이지 트리 (parent_id 자기참조) | 4/26 |
| M3 | 권한: Notion 식 상속 (Phase 2 page_acl override) | 4/26 |
| M4 | 채널 ↔ 페이지 다대다 (자동 부착 안 함) | 4/26 |
| M5 | Pages = boards.layout='doc' + blocks.type='page' (대규모 새 테이블 미사용) | 4/26 |
| M6 | 코드 위치 분리: `webapp/src/pages/`, `server/pages/` 등 | 4/26 |
| M7 | focalboard 본가 머지: 보안 패치만 cherry-pick | 4/26 |

### 세부 결정 (Q1-Q13)

| Q | 결정 |
|---|---|
| Q1.1 사이클 방지 | 서버 검증 + UI 에러 |
| Q1.2 트리 최대 깊이 | 10단계 |
| Q1.3 부모당 자식 수 | 1000 + 사이드바 가상화 |
| Q2.1 삭제 시 자식 처리 | **컨펌 다이얼로그 (cascade or 부모로 승격 선택)** |
| Q2.1 깊이 위반 처리 | 경고 + 진행 |
| Q2.2 휴지통 복원 윈도우 | 30일 |
| Q2.3 빈 doc-board 정리 | 명시 삭제만 |
| Q3.1 URL 형식 | `/page/{slug}-{shortid}` |
| Q3.1.a 한국어 처리 | IRI 그대로 |
| Q3.1.b shortid 길이 | 8자 |
| Q3.1.c slug 길이 제한 | 50자 |
| Q3.1.d 제목 변경 시 옛 URL | 301 리다이렉트 |
| Q3.2 멘션 자동 갱신 | 자동 (ID 저장, 렌더 시 lookup) |
| Q3.3 백링크 | Phase 2 |
| Q3.4 블록 anchor 링크 | 헤딩만 |
| Q4.1 Markdown export | Phase 1 |
| Q4.2 PDF export | Phase 1 (브라우저 print) → Phase 3 (서버사이드) |
| Q4.3 책 단위 export | Phase 3 |
| Q5.1 한국어 IME 검증 | 본 작업 전 PoC |
| Q6.1 본가 머지 정책 | 보안 패치만 cherry-pick |
| Q6.2 코드 네임스페이스 | 별도 디렉토리 |
| Q7.1 검색 권한 필터링 | 통과 페이지만 결과 |
| Q7.2 검색 범위 | 워크스페이스 / 책 토글 |
| Q8.1 Yjs 압축 | 주기 스냅샷 + 옛 update 삭제 |
| Q8.2 압축 주기 | 5분 / 100 updates |
| Q8.3 Awareness ghost | 30초 timeout |
| Q9.1 멘션 알림 | MM 알림 시스템 후크 |
| Q9.2 페이지 구독 | Phase 3 |
| Q9.3 코멘트 resolve | Phase 3 |
| Q10.1 XSS | Tiptap 스키마 + DOMPurify |
| Q10.2 WS 권한 재검사 | 매번 |
| Q10.3 파일 업로드 크기 | MM 설정 따름 |
| Q11.1 모바일 편집 | 읽기 + 텍스트 편집만 (객체 추가 불가) |
| Q12.1 접근성 | Phase 4 |
| Q13.1 Markdown import | Phase 1 |
| Q13.2 Confluence/Notion import | 안 함 |

---

## 12. 미해결 위험 / 의존성

| 위험 | 영향 | 완화 |
|---|---|---|
| **한국어 IME 호환성** | Phase 1 에 들어가서야 발견되면 epic 분기 | **본 작업 전 1-2일 PoC 필수**. 막히면 Lexical 등 대안 평가 |
| **focalboard upstream 큰 리팩터** | 본가가 blocks 시스템을 갈아엎으면 우리 fork 가 깊이 손상 | 보안 패치만 cherry-pick 정책으로 일부 완화. 큰 변경은 그때 의사결정 |
| **Yjs DB 비대화** | update 누적 또는 압축 실패 시 page_yjs_updates 폭증 | 압축 모니터링 + 알림. Phase 2 출시 직후 운영 지표 점검 |
| **MM WebSocket 메시지 크기 한계** | Yjs binary update 가 큰 경우 (대량 paste 등) WS 메시지 한계 초과 | chunking 또는 fallback HTTP POST |
| **트리 동시 변경 충돌** | 두 사용자가 동시에 같은 페이지 이동 | 서버에서 atomic 트랜잭션 + 충돌 시 후-도착 거부 |
| **디스크** | 현재 / 99% 사용. 빌드 가능한지 매번 점검 필요 | `go clean -modcache` 등으로 ~3.9GB 회수 가능 |

---

## 13. Phase 1 완료 기준 (Definition of Done)

- [ ] doc-board 생성 가능 (`layout='doc'` 토글)
- [ ] 페이지 생성 / 이동 / 삭제 (cascade or 승격)
- [ ] 트리 사이드바 — 접기/펼치기, 드래그 정렬
- [ ] Tiptap 에디터 — 헤딩 / 단락 / 인라인 스타일 / 리스트 (불릿/번호/체크) / 표 / 코드 블록 / 인용 / 콜아웃 / 구분선 / 이미지 / 링크 / @멘션
- [ ] 한국어 IME 정상 동작 (PoC 결과 반영)
- [ ] 페이지 권한 — board_members 식 (Boards 와 동일 모델)
- [ ] URL 형식 `/page/{slug}-{shortid}`, 한국어 IRI 동작
- [ ] Markdown export / import
- [ ] 검색 (권한 필터링)
- [ ] 단일 편집자 모드 (낙관적 락, Yjs 는 Phase 2)
- [ ] i18n: ko 기본
- [ ] 기존 Boards 기능 회귀 없음

---

## 14. 다음 단계

1. **한국어 IME PoC** (1-2일) — Tiptap 기본 셋업 + composition event 검증
2. **DB 마이그레이션 SQL 초안 작성** — 위 3.1, 3.2 의 ALTER 와 CREATE TABLE 들을 마이그레이션 파일로
3. **Go 모델 정의** — Page 구조체, BlockTypePage 상수, Board.Layout 필드 등
4. **TS 모델 정의** — webapp 측 타입 + Redux slice 스켈레톤
5. **Tiptap 노드 셋팅** — 위 13 의 노드 타입 전부

---

## 부록 A — 재사용 매트릭스 (Boards 인프라 → Pages)

| Boards 영역 | Pages 가 어떻게 사용하나 |
|---|---|
| `boards` 테이블 | doc-board 도 같은 테이블 (layout='doc') |
| `blocks` 테이블 | 페이지를 type='page' row 로 저장 (메타만, 콘텐츠는 page_content) |
| `boards_history` | doc-board 변경 히스토리에 그대로 활용 |
| `board_members` | doc-board 멤버십 그대로 — 페이지 권한 1차 ACL |
| `Sharing` 토큰 | 페이지 공개 공유에 그대로 |
| `mmpermissions.HasPermissionToBoard` | doc-board 에 그대로 동작. Pages 는 그 위에 page_acl 추가 검사 |
| Synthetic membership | 채널 링크된 doc-board 에 그대로 동작 |
| MinimumRole | doc-board 에 그대로 |
| Team Admin elevation | 그대로 |
| `wsAdapter` (WebSocket) | Yjs binary update 송수신 채널로 활용 |
| `cardDetailContents.tsx` | (재사용 안 함, 페이지는 별도 PageEditor) |
| 사이드바 카테고리 | 그대로 활용, doc-board 만 트리 모드로 분기 |
| 검색 인프라 | blocks 검색 그대로 + page_content 검색 추가 |
| Redux blocks slice | 페이지 메타데이터에 그대로 활용, 콘텐츠는 별도 pages slice |

---

## 부록 B — 신규 의존 라이브러리

| 패키지 | 용도 | 라이선스 |
|---|---|---|
| `@tiptap/core`, `@tiptap/starter-kit`, `@tiptap/extension-*` | 에디터 | MIT |
| `yjs` | CRDT 코어 | MIT |
| `y-prosemirror` | Tiptap ↔ Yjs 바인딩 | MIT |
| `y-protocols` | Yjs awareness/sync | MIT |
| `nanoid` | shortid 생성 | MIT |
| `dompurify` | XSS 정화 | Apache-2.0 |
| `marked` 또는 `remark` | Markdown export/import | MIT |

Go 측: 표준 lib + 기존 focalboard 의존만 사용. Yjs Go 포트는 미도입 (서버는 패시브 릴레이).

---

**문서 끝.** 다음 회의/턴에서 Phase 1 착수 전 한국어 IME PoC 일정 + DB 마이그레이션 SQL 초안부터 진행하겠습니다.
