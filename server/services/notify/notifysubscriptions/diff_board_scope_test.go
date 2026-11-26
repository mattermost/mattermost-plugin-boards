// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package notifysubscriptions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-boards/server/model"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// historyStore serves block history from memory. Each block has a single
// version; children are returned by parent ID only, like the SQL store.
type historyStore struct {
	blocks map[string]*model.Block
}

func (s *historyStore) GetBlockHistory(blockID string, opts model.QueryBlockHistoryOptions) ([]*model.Block, error) {
	b, ok := s.blocks[blockID]
	if !ok {
		return nil, nil
	}
	if opts.BeforeUpdateAt != 0 && b.UpdateAt >= opts.BeforeUpdateAt {
		return nil, nil
	}
	if opts.AfterUpdateAt != 0 && b.UpdateAt <= opts.AfterUpdateAt {
		return nil, nil
	}
	return []*model.Block{b}, nil
}

func (s *historyStore) GetBlockHistoryNewestChildren(parentID string, opts model.QueryBlockHistoryChildOptions) ([]*model.Block, bool, error) {
	var children []*model.Block
	for _, b := range s.blocks {
		if b.ParentID != parentID {
			continue
		}
		if opts.AfterUpdateAt != 0 && b.UpdateAt <= opts.AfterUpdateAt {
			continue
		}
		children = append(children, b)
	}
	return children, false, nil
}

func (s *historyStore) GetBoardAndCardByID(string) (*model.Board, *model.Block, error) {
	return nil, nil, nil
}

func (s *historyStore) GetUserByID(userID string) (*model.User, error) {
	return &model.User{ID: userID, Username: userID}, nil
}

func (s *historyStore) CreateSubscription(sub *model.Subscription) (*model.Subscription, error) {
	return sub, nil
}

func (s *historyStore) GetSubscribersForBlock(string) ([]*model.Subscriber, error) {
	return nil, nil
}

func (s *historyStore) UpdateSubscribersNotifiedAt(string, int64) error {
	return nil
}

func (s *historyStore) UpsertNotificationHint(hint *model.NotificationHint, _ time.Duration) (*model.NotificationHint, error) {
	return hint, nil
}

func (s *historyStore) GetNextNotificationHint(bool) (*model.NotificationHint, error) {
	return nil, nil
}

// A block stored with a parent on another board (data written before parents
// were validated) must not show up in that card's change notifications.
func TestGenerateDiffsForCardOnlyIncludesChildrenFromCardBoard(t *testing.T) {
	boardB := &model.Board{ID: "board-b", TeamID: "team-id", Title: "Board B"}
	cardB := &model.Block{
		ID: "card-b", BoardID: boardB.ID, ParentID: boardB.ID, Type: model.TypeCard,
		Title: "card B", ModifiedBy: "user-b", UpdateAt: 200, Fields: map[string]interface{}{},
	}
	textB := &model.Block{
		ID: "text-b", BoardID: boardB.ID, ParentID: cardB.ID, Type: model.TypeText,
		Title: "text on card B", ModifiedBy: "user-b", UpdateAt: 200, Fields: map[string]interface{}{},
	}
	commentA := &model.Block{
		ID: "comment-a", BoardID: "board-a", ParentID: cardB.ID, Type: model.TypeComment,
		Title: "comment stored on board A", ModifiedBy: "user-a", UpdateAt: 200, Fields: map[string]interface{}{},
	}

	store := &historyStore{blocks: map[string]*model.Block{
		cardB.ID:    cardB,
		textB.ID:    textB,
		commentA.ID: commentA,
	}}

	dg := &diffGenerator{
		board:        boardB,
		card:         cardB,
		store:        store,
		hint:         &model.NotificationHint{BlockType: model.TypeCard, BlockID: cardB.ID},
		lastNotifyAt: 100,
		logger:       mlog.CreateConsoleTestLogger(t),
	}

	diffs, err := dg.generateDiffs()
	require.NoError(t, err)
	require.Len(t, diffs, 1)

	childIDs := make([]string, 0, len(diffs[0].Diffs))
	for _, d := range diffs[0].Diffs {
		require.NotNil(t, d.NewBlock)
		childIDs = append(childIDs, d.NewBlock.ID)
	}
	require.Contains(t, childIDs, textB.ID)
	require.NotContains(t, childIDs, commentA.ID, "child block from another board included in card notification")
	require.NotContains(t, diffs[0].Authors.Values(), "user-a", "author of a block on another board listed in card notification")
}
