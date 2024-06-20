package app

import (
	"errors"
	"fmt"

	"github.com/mattermost/mattermost-plugin-boards/server/model"
	"github.com/mattermost/mattermost-plugin-boards/server/services/notify"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

var ErrBlocksFromMultipleBoards = errors.New("the block set contain blocks from multiple boards")

func (a *App) GetBlocks(boardID, parentID string, blockType string) ([]*model.Block, error) {
	if boardID == "" {
		return []*model.Block{}, nil
	}

	if blockType != "" && parentID != "" {
		return a.store.GetBlocksWithParentAndType(boardID, parentID, blockType)
	}

	if blockType != "" {
		return a.store.GetBlocksWithType(boardID, blockType)
	}

	return a.store.GetBlocksWithParent(boardID, parentID)
}

func (a *App) DuplicateBlock(boardID string, blockID string, userID string, asTemplate bool) ([]*model.Block, error) {
	board, err := a.GetBoard(boardID)
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, fmt.Errorf("cannot fetch board %s for DuplicateBlock: %w", boardID, err)
	}

	blocks, err := a.store.DuplicateBlock(boardID, blockID, userID, asTemplate)
	if err != nil {
		return nil, err
	}

	err = a.CopyAndUpdateCardFiles(boardID, userID, blocks, asTemplate)
	if err != nil {
		return nil, err
	}

	a.blockChangeNotifier.Enqueue(func() error {
		for _, block := range blocks {
			a.wsAdapter.BroadcastBlockChange(board.TeamID, block)
		}
		return nil
	})

	return blocks, err
}

func (a *App) PatchBlock(blockID string, blockPatch *model.BlockPatch, modifiedByID string) (*model.Block, error) {
	return a.PatchBlockAndNotify(blockID, blockPatch, modifiedByID, false)
}

func (a *App) PatchBlockAndNotify(blockID string, blockPatch *model.BlockPatch, modifiedByID string, disableNotify bool) (*model.Block, error) {
	oldBlock, err := a.store.GetBlock(blockID)
	if err != nil {
		return nil, err
	}

	board, err := a.store.GetBoard(oldBlock.BoardID)
	if err != nil {
		return nil, err
	}

	err = a.store.PatchBlock(blockID, blockPatch, modifiedByID)
	if err != nil {
		return nil, err
	}

	a.metrics.IncrementBlocksPatched(1)
	block, err := a.store.GetBlock(blockID)
	if err != nil {
		return nil, err
	}
	a.blockChangeNotifier.Enqueue(func() error {
		// broadcast on websocket
		a.wsAdapter.BroadcastBlockChange(board.TeamID, block)

		// broadcast on webhooks
		a.webhook.NotifyUpdate(block)

		// send notifications
		if !disableNotify {
			a.notifyBlockChanged(notify.Update, block, oldBlock, modifiedByID)
		}
		return nil
	})
	return block, nil
}

func (a *App) PatchBlocks(teamID string, blockPatches *model.BlockPatchBatch, modifiedByID string) error {
	return a.PatchBlocksAndNotify(teamID, blockPatches, modifiedByID, false)
}

func (a *App) PatchBlocksAndNotify(teamID string, blockPatches *model.BlockPatchBatch, modifiedByID string, disableNotify bool) error {
	oldBlocks, err := a.store.GetBlocksByIDs(blockPatches.BlockIDs)
	if err != nil {
		return err
	}

	if err := a.store.PatchBlocks(blockPatches, modifiedByID); err != nil {
		return err
	}

	a.blockChangeNotifier.Enqueue(func() error {
		a.metrics.IncrementBlocksPatched(len(oldBlocks))
		for i, blockID := range blockPatches.BlockIDs {
			newBlock, err := a.store.GetBlock(blockID)
			if err != nil {
				return err
			}
			a.wsAdapter.BroadcastBlockChange(teamID, newBlock)
			a.webhook.NotifyUpdate(newBlock)
			if !disableNotify {
				a.notifyBlockChanged(notify.Update, newBlock, oldBlocks[i], modifiedByID)
			}
		}
		return nil
	})
	return nil
}

func (a *App) InsertBlock(block *model.Block, modifiedByID string) error {
	return a.InsertBlockAndNotify(block, modifiedByID, false)
}

func (a *App) InsertBlockAndNotify(block *model.Block, modifiedByID string, disableNotify bool) error {
	board, bErr := a.store.GetBoard(block.BoardID)
	if bErr != nil {
		return bErr
	}

	err := a.store.InsertBlock(block, modifiedByID)
	if err == nil {
		a.blockChangeNotifier.Enqueue(func() error {
			a.wsAdapter.BroadcastBlockChange(board.TeamID, block)
			a.metrics.IncrementBlocksInserted(1)
			a.webhook.NotifyUpdate(block)
			if !disableNotify {
				a.notifyBlockChanged(notify.Add, block, nil, modifiedByID)
			}
			return nil
		})
	}

	return err
}

func (a *App) InsertBlocks(blocks []*model.Block, modifiedByID string) ([]*model.Block, error) {
	return a.InsertBlocksAndNotify(blocks, modifiedByID, false)
}

func (a *App) InsertBlocksAndNotify(blocks []*model.Block, modifiedByID string, disableNotify bool) ([]*model.Block, error) {
	if len(blocks) == 0 {
		return []*model.Block{}, nil
	}

	// all blocks must belong to the same board
	boardID := blocks[0].BoardID
	for _, block := range blocks {
		if block.BoardID != boardID {
			return nil, ErrBlocksFromMultipleBoards
		}
	}

	board, err := a.store.GetBoard(boardID)
	if err != nil {
		return nil, err
	}

	needsNotify := make([]*model.Block, 0, len(blocks))
	for i := range blocks {
		err := a.store.InsertBlock(blocks[i], modifiedByID)
		if err != nil {
			return nil, err
		}
		needsNotify = append(needsNotify, blocks[i])

		a.wsAdapter.BroadcastBlockChange(board.TeamID, blocks[i])
		a.metrics.IncrementBlocksInserted(1)
	}

	a.blockChangeNotifier.Enqueue(func() error {
		for _, b := range needsNotify {
			block := b
			a.webhook.NotifyUpdate(block)
			if !disableNotify {
				a.notifyBlockChanged(notify.Add, block, nil, modifiedByID)
			}
		}
		return nil
	})

	return blocks, nil
}

func (a *App) GetBlockByID(blockID string) (*model.Block, error) {
	return a.store.GetBlock(blockID)
}

func (a *App) DeleteBlock(blockID string, modifiedBy string) error {
	return a.DeleteBlockAndNotify(blockID, modifiedBy, false)
}

func (a *App) DeleteBlockAndNotify(blockID string, modifiedBy string, disableNotify bool) error {
	block, err := a.store.GetBlock(blockID)
	if err != nil {
		return err
	}

	board, err := a.store.GetBoard(block.BoardID)
	if err != nil {
		return err
	}

	if block == nil {
		// deleting non-existing block not considered an error
		return nil
	}

	err = a.store.DeleteBlock(blockID, modifiedBy)
	if err != nil {
		return err
	}

	a.blockChangeNotifier.Enqueue(func() error {
		a.wsAdapter.BroadcastBlockDelete(board.TeamID, blockID, block.BoardID)
		a.metrics.IncrementBlocksDeleted(1)
		if !disableNotify {
			a.notifyBlockChanged(notify.Delete, block, block, modifiedBy)
		}
		return nil
	})

	return nil
}

func (a *App) GetLastBlockHistoryEntry(blockID string) (*model.Block, error) {
	blocks, err := a.store.GetBlockHistory(blockID, model.QueryBlockHistoryOptions{Limit: 1, Descending: true})
	if err != nil {
		return nil, err
	}
	if len(blocks) == 0 {
		return nil, nil
	}
	return blocks[0], nil
}

func (a *App) UndeleteBlock(blockID string, modifiedBy string) (*model.Block, error) {
	blocks, err := a.store.GetBlockHistory(blockID, model.QueryBlockHistoryOptions{Limit: 1, Descending: true})
	if err != nil {
		return nil, err
	}

	if len(blocks) == 0 {
		// undeleting non-existing block not considered an error
		return nil, nil
	}

	err = a.store.UndeleteBlock(blockID, modifiedBy)
	if err != nil {
		return nil, err
	}

	block, err := a.store.GetBlock(blockID)
	if model.IsErrNotFound(err) {
		a.logger.Error("Error loading the block after a successful undelete, not propagating through websockets or notifications", mlog.String("blockID", blockID))
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	board, err := a.store.GetBoard(block.BoardID)
	if err != nil {
		return nil, err
	}

	a.blockChangeNotifier.Enqueue(func() error {
		a.wsAdapter.BroadcastBlockChange(board.TeamID, block)
		a.metrics.IncrementBlocksInserted(1)
		a.webhook.NotifyUpdate(block)
		a.notifyBlockChanged(notify.Add, block, nil, modifiedBy)

		return nil
	})

	return block, nil
}

func (a *App) GetBlockCountsByType() (map[string]int64, error) {
	return a.store.GetBlockCountsByType()
}

func (a *App) GetBlocksForBoard(boardID string) ([]*model.Block, error) {
	return a.store.GetBlocksForBoard(boardID)
}

func (a *App) notifyBlockChanged(action notify.Action, block *model.Block, oldBlock *model.Block, modifiedByID string) {
	// don't notify if notifications service disabled, or block change is generated via system user.
	if a.notifications == nil || modifiedByID == model.SystemUserID {
		return
	}

	// find card and board for the changed block.
	board, card, err := a.getBoardAndCard(block)
	if err != nil {
		a.logger.Error("Error notifying for block change; cannot determine board or card", mlog.Err(err))
		return
	}

	boardMember, _ := a.GetMemberForBoard(board.ID, modifiedByID)
	if boardMember == nil {
		// create temporary guest board member
		boardMember = &model.BoardMember{
			BoardID: board.ID,
			UserID:  modifiedByID,
		}
	}

	evt := notify.BlockChangeEvent{
		Action:       action,
		TeamID:       board.TeamID,
		Board:        board,
		Card:         card,
		BlockChanged: block,
		BlockOld:     oldBlock,
		ModifiedBy:   boardMember,
	}
	a.notifications.BlockChanged(evt)
}

const (
	maxSearchDepth = 50
)

// getBoardAndCard returns the first parent of type `card` its board for the specified block.
// `board` and/or `card` may return nil without error if the block does not belong to a board or card.
func (a *App) getBoardAndCard(block *model.Block) (board *model.Board, card *model.Block, err error) {
	board, err = a.store.GetBoard(block.BoardID)
	if err != nil {
		return board, card, err
	}

	var count int // don't let invalid blocks hierarchy cause infinite loop.
	iter := block
	for {
		count++
		if card == nil && iter.Type == model.TypeCard {
			card = iter
		}

		if iter.ParentID == "" || (board != nil && card != nil) || count > maxSearchDepth {
			break
		}

		iter, err = a.store.GetBlock(iter.ParentID)
		if model.IsErrNotFound(err) {
			return board, card, nil
		}
		if err != nil {
			return board, card, err
		}
	}
	return board, card, nil
}
