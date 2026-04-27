# VS12 — Doc-Board 진입점 (Phase 1 vertical slice)

> **연관**: [PAGES_CHECKLIST.md](../PAGES_CHECKLIST.md), [PAGES_PLAN.md](../PAGES_PLAN.md)

Phase 1 의 vertical slice 검증을 위해 doc-board 를 만드는 가장 빠른 길.

## 진입점 (3가지, 빠른 순)

### 1) curl 로 doc-board 만들고 URL 직접 진입 (1차 vertical slice 검증용)

#### 1-1. 로그인 토큰 얻기
```bash
source /root/mattermost-plugin-calls/.env

TOKEN=$(curl -si -X POST "$SERVER_URL/api/v4/users/login" \
  -H 'Content-Type: application/json' \
  -d "{\"login_id\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}" \
  | grep -i '^token:' | awk '{print $2}' | tr -d '\r\n')
```

#### 1-2. 현재 사용자의 team 확인
```bash
TEAM_ID=$(curl -s -H "Authorization: Bearer $TOKEN" \
  "$SERVER_URL/api/v4/users/me/teams" \
  | python3 -c "import sys, json; print(json.load(sys.stdin)[0]['id'])")
echo "TEAM_ID=$TEAM_ID"
```

#### 1-3. doc-board 생성 (focalboard board 생성 + layout='doc')

```bash
BOARD=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  "$SERVER_URL/plugins/focalboard/api/v2/boards" \
  -d "$(cat <<EOF
{
  "teamId": "$TEAM_ID",
  "title": "신입 온보딩 매뉴얼",
  "type": "P",
  "minimumRole": "viewer",
  "layout": "doc"
}
EOF
)")
BOARD_ID=$(echo "$BOARD" | python3 -c "import sys, json; print(json.load(sys.stdin)['id'])")
echo "BOARD_ID=$BOARD_ID"
```

#### 1-4. 브라우저에서 진입
```
http://localhost:8065/boards/team/{TEAM_ID}/doc/{BOARD_ID}
```

(focalboard 가 `/boards/` prefix 아래 마운트됨. 정확한 prefix 는 plugin manifest 확인)

### 2) Pages API 로 페이지 직접 생성 (서버 단독 검증)

```bash
PAGE=$(curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  "$SERVER_URL/plugins/focalboard/api/v2/boards/$BOARD_ID/pages" \
  -d '{
    "parentId": "'$BOARD_ID'",
    "title": "1주차 가이드"
  }')
PAGE_ID=$(echo "$PAGE" | python3 -c "import sys, json; print(json.load(sys.stdin)['id'])")
echo "PAGE_ID=$PAGE_ID"
```

브라우저:
```
http://localhost:8065/boards/team/{TEAM_ID}/doc/{BOARD_ID}/{PAGE_ID}
```

### 3) 사이드바에 진입점 통합 (Phase 1 후속, vertical slice 후 추가 작업)

`webapp/src/components/sidebar/` 의 보드 목록 렌더에 분기 추가:
- 보드의 `layout === 'doc'` 이면 클릭 시 `/team/.../doc/...` 로 라우팅
- 그렇지 않으면 기존 `/team/.../boardId/...` 로

이는 fragile wiring 이라 별도 PR 로 분리 권장.

## Phase 1 후속 — UI 진입점 후보들

| 후보 | 위치 | 작업량 |
|---|---|---|
| 사이드바에서 layout='doc' 자동 분기 | `sidebar.tsx` 의 click handler | 1-2시간 |
| "+ Add board" 다이얼로그에 layout 토글 | `BoardTemplateSelector` | 2-3시간 |
| 기존 보드를 doc 로 변환 메뉴 | board settings | 1-2시간 |
| 채널 헤더에 "Pages" 버튼 | header 컴포넌트 | 2-3시간 |

## 디버깅

서버 로그에서 페이지 API 호출 추적:
```bash
tail -f /root/mattermost/server/logs/mattermost.log | grep -E "createPage|pages/"
```

DB 직접 확인:
```bash
# Postgres 가정
psql -d mattermost -c "
  SELECT id, title, layout FROM focalboard_boards WHERE layout = 'doc';
"
psql -d mattermost -c "
  SELECT id, parent_id, title FROM focalboard_blocks WHERE type = 'page';
"
psql -d mattermost -c "
  SELECT page_id, length(tiptap_json::text), update_at FROM focalboard_page_content;
"
```
