// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"testing"

	mmModel "github.com/mattermost/mattermost/server/public/model"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost/server/public/shared/mlog"

	"github.com/mattermost/mattermost-plugin-boards/server/utils"

	"github.com/stretchr/testify/require"
)

func TestGenerateBlockIDs(t *testing.T) {
	t.Run("Should generate a new ID for a single block with no references", func(t *testing.T) {
		blockID := utils.NewID(utils.IDTypeBlock)
		blocks := []*Block{{ID: blockID}}

		blocks = GenerateBlockIDs(blocks, &mlog.Logger{})

		require.NotEqual(t, blockID, blocks[0].ID)
		require.Zero(t, blocks[0].BoardID)
		require.Zero(t, blocks[0].ParentID)
	})

	t.Run("Should generate a new ID for a single block with references", func(t *testing.T) {
		blockID := utils.NewID(utils.IDTypeBlock)
		boardID := utils.NewID(utils.IDTypeBlock)
		parentID := utils.NewID(utils.IDTypeBlock)
		blocks := []*Block{{ID: blockID, BoardID: boardID, ParentID: parentID}}

		blocks = GenerateBlockIDs(blocks, &mlog.Logger{})

		require.NotEqual(t, blockID, blocks[0].ID)
		require.Equal(t, boardID, blocks[0].BoardID)
		require.Equal(t, parentID, blocks[0].ParentID)
	})

	t.Run("Should generate IDs and link multiple blocks with existing references", func(t *testing.T) {
		blockID1 := utils.NewID(utils.IDTypeBlock)
		boardID1 := utils.NewID(utils.IDTypeBlock)
		parentID1 := utils.NewID(utils.IDTypeBlock)
		block1 := &Block{ID: blockID1, BoardID: boardID1, ParentID: parentID1}

		blockID2 := utils.NewID(utils.IDTypeBlock)
		boardID2 := blockID1
		parentID2 := utils.NewID(utils.IDTypeBlock)
		block2 := &Block{ID: blockID2, BoardID: boardID2, ParentID: parentID2}

		blocks := []*Block{block1, block2}

		blocks = GenerateBlockIDs(blocks, &mlog.Logger{})

		require.NotEqual(t, blockID1, blocks[0].ID)
		require.Equal(t, boardID1, blocks[0].BoardID)
		require.Equal(t, parentID1, blocks[0].ParentID)

		require.NotEqual(t, blockID2, blocks[1].ID)
		require.NotEqual(t, boardID2, blocks[1].BoardID)
		require.Equal(t, parentID2, blocks[1].ParentID)

		// blockID1 was referenced, so it should still be after the ID
		// changes
		require.Equal(t, blocks[0].ID, blocks[1].BoardID)
	})

	t.Run("Should generate new IDs but not modify nonexisting references", func(t *testing.T) {
		blockID1 := utils.NewID(utils.IDTypeBlock)
		boardID1 := ""
		parentID1 := utils.NewID(utils.IDTypeBlock)
		block1 := &Block{ID: blockID1, BoardID: boardID1, ParentID: parentID1}

		blockID2 := utils.NewID(utils.IDTypeBlock)
		boardID2 := utils.NewID(utils.IDTypeBlock)
		parentID2 := ""
		block2 := &Block{ID: blockID2, BoardID: boardID2, ParentID: parentID2}

		blocks := []*Block{block1, block2}

		blocks = GenerateBlockIDs(blocks, &mlog.Logger{})

		// only the IDs should have changed
		require.NotEqual(t, blockID1, blocks[0].ID)
		require.Zero(t, blocks[0].BoardID)
		require.Equal(t, parentID1, blocks[0].ParentID)

		require.NotEqual(t, blockID2, blocks[1].ID)
		require.Equal(t, boardID2, blocks[1].BoardID)
		require.Zero(t, blocks[1].ParentID)
	})

	t.Run("Should modify correctly multiple blocks with existing and nonexisting references", func(t *testing.T) {
		blockID1 := utils.NewID(utils.IDTypeBlock)
		boardID1 := utils.NewID(utils.IDTypeBlock)
		parentID1 := utils.NewID(utils.IDTypeBlock)
		block1 := &Block{ID: blockID1, BoardID: boardID1, ParentID: parentID1}

		// linked to 1
		blockID2 := utils.NewID(utils.IDTypeBlock)
		boardID2 := blockID1
		parentID2 := utils.NewID(utils.IDTypeBlock)
		block2 := &Block{ID: blockID2, BoardID: boardID2, ParentID: parentID2}

		// linked to 2
		blockID3 := utils.NewID(utils.IDTypeBlock)
		boardID3 := blockID2
		parentID3 := utils.NewID(utils.IDTypeBlock)
		block3 := &Block{ID: blockID3, BoardID: boardID3, ParentID: parentID3}

		// linked to 1
		blockID4 := utils.NewID(utils.IDTypeBlock)
		boardID4 := blockID1
		parentID4 := utils.NewID(utils.IDTypeBlock)
		block4 := &Block{ID: blockID4, BoardID: boardID4, ParentID: parentID4}

		// blocks are shuffled
		blocks := []*Block{block4, block2, block1, block3}

		blocks = GenerateBlockIDs(blocks, &mlog.Logger{})

		// block 1
		require.NotEqual(t, blockID1, blocks[2].ID)
		require.Equal(t, boardID1, blocks[2].BoardID)
		require.Equal(t, parentID1, blocks[2].ParentID)

		// block 2
		require.NotEqual(t, blockID2, blocks[1].ID)
		require.NotEqual(t, boardID2, blocks[1].BoardID)
		require.Equal(t, blocks[2].ID, blocks[1].BoardID) // link to 1
		require.Equal(t, parentID2, blocks[1].ParentID)

		// block 3
		require.NotEqual(t, blockID3, blocks[3].ID)
		require.NotEqual(t, boardID3, blocks[3].BoardID)
		require.Equal(t, blocks[1].ID, blocks[3].BoardID) // link to 2
		require.Equal(t, parentID3, blocks[3].ParentID)

		// block 4
		require.NotEqual(t, blockID4, blocks[0].ID)
		require.NotEqual(t, boardID4, blocks[0].BoardID)
		require.Equal(t, blocks[2].ID, blocks[0].BoardID) // link to 1
		require.Equal(t, parentID4, blocks[0].ParentID)
	})

	t.Run("Should update content order", func(t *testing.T) {
		blockID1 := utils.NewID(utils.IDTypeBlock)
		boardID1 := utils.NewID(utils.IDTypeBlock)
		parentID1 := utils.NewID(utils.IDTypeBlock)
		block1 := &Block{
			ID:       blockID1,
			BoardID:  boardID1,
			ParentID: parentID1,
		}

		blockID2 := utils.NewID(utils.IDTypeBlock)
		boardID2 := utils.NewID(utils.IDTypeBlock)
		parentID2 := utils.NewID(utils.IDTypeBlock)
		block2 := &Block{
			ID:       blockID2,
			BoardID:  boardID2,
			ParentID: parentID2,
			Fields: map[string]interface{}{
				"contentOrder": []interface{}{
					blockID1,
				},
			},
		}

		blocks := []*Block{block1, block2}

		blocks = GenerateBlockIDs(blocks, &mlog.Logger{})

		require.NotEqual(t, blockID1, blocks[0].ID)
		require.Equal(t, boardID1, blocks[0].BoardID)
		require.Equal(t, parentID1, blocks[0].ParentID)

		require.NotEqual(t, blockID2, blocks[1].ID)
		require.Equal(t, boardID2, blocks[1].BoardID)
		require.Equal(t, parentID2, blocks[1].ParentID)

		// since block 1 was referenced in block 2,
		// the ID should have been changed in content order
		block2ContentOrder, ok := block2.Fields["contentOrder"].([]interface{})
		require.True(t, ok)
		require.NotEqual(t, blockID1, block2ContentOrder[0].(string))
		require.Equal(t, blocks[0].ID, block2ContentOrder[0].(string))
	})

	t.Run("Should update content order when it contain slices", func(t *testing.T) {
		blockID1 := utils.NewID(utils.IDTypeBlock)
		boardID1 := utils.NewID(utils.IDTypeBlock)
		parentID1 := utils.NewID(utils.IDTypeBlock)
		block1 := &Block{
			ID:       blockID1,
			BoardID:  boardID1,
			ParentID: parentID1,
		}

		blockID2 := utils.NewID(utils.IDTypeBlock)
		block2 := &Block{
			ID:       blockID2,
			BoardID:  boardID1,
			ParentID: parentID1,
		}

		blockID3 := utils.NewID(utils.IDTypeBlock)
		block3 := &Block{
			ID:       blockID3,
			BoardID:  boardID1,
			ParentID: parentID1,
		}

		blockID4 := utils.NewID(utils.IDTypeBlock)
		boardID2 := utils.NewID(utils.IDTypeBlock)
		parentID2 := utils.NewID(utils.IDTypeBlock)

		block4 := &Block{
			ID:       blockID4,
			BoardID:  boardID2,
			ParentID: parentID2,
			Fields: map[string]interface{}{
				"contentOrder": []interface{}{
					blockID1,
					[]interface{}{
						blockID2,
						blockID3,
					},
				},
			},
		}

		blocks := []*Block{block1, block2, block3, block4}

		blocks = GenerateBlockIDs(blocks, &mlog.Logger{})

		require.NotEqual(t, blockID1, blocks[0].ID)
		require.Equal(t, boardID1, blocks[0].BoardID)
		require.Equal(t, parentID1, blocks[0].ParentID)

		require.NotEqual(t, blockID4, blocks[3].ID)
		require.Equal(t, boardID2, blocks[3].BoardID)
		require.Equal(t, parentID2, blocks[3].ParentID)

		// since block 1 was referenced in block 2,
		// the ID should have been changed in content order
		block4ContentOrder, ok := block4.Fields["contentOrder"].([]interface{})
		require.True(t, ok)
		require.NotEqual(t, blockID1, block4ContentOrder[0].(string))
		require.NotEqual(t, blockID2, block4ContentOrder[1].([]interface{})[0])
		require.NotEqual(t, blockID3, block4ContentOrder[1].([]interface{})[1])
		require.Equal(t, blocks[0].ID, block4ContentOrder[0].(string))
		require.Equal(t, blocks[1].ID, block4ContentOrder[1].([]interface{})[0])
		require.Equal(t, blocks[2].ID, block4ContentOrder[1].([]interface{})[1])
	})

	t.Run("Should update Id of default template view", func(t *testing.T) {
		blockID1 := utils.NewID(utils.IDTypeBlock)
		boardID1 := utils.NewID(utils.IDTypeBlock)
		parentID1 := utils.NewID(utils.IDTypeBlock)
		block1 := &Block{
			ID:       blockID1,
			BoardID:  boardID1,
			ParentID: parentID1,
		}

		blockID2 := utils.NewID(utils.IDTypeBlock)
		boardID2 := utils.NewID(utils.IDTypeBlock)
		parentID2 := utils.NewID(utils.IDTypeBlock)
		block2 := &Block{
			ID:       blockID2,
			BoardID:  boardID2,
			ParentID: parentID2,
			Fields: map[string]interface{}{
				"defaultTemplateId": blockID1,
			},
		}

		blocks := []*Block{block1, block2}

		blocks = GenerateBlockIDs(blocks, &mlog.Logger{})

		require.NotEqual(t, blockID1, blocks[0].ID)
		require.Equal(t, boardID1, blocks[0].BoardID)
		require.Equal(t, parentID1, blocks[0].ParentID)

		require.NotEqual(t, blockID2, blocks[1].ID)
		require.Equal(t, boardID2, blocks[1].BoardID)
		require.Equal(t, parentID2, blocks[1].ParentID)

		block2DefaultTemplateID, ok := block2.Fields["defaultTemplateId"].(string)
		require.True(t, ok)
		require.NotEqual(t, blockID1, block2DefaultTemplateID)
		require.Equal(t, blocks[0].ID, block2DefaultTemplateID)
	})
}

func TestStampModificationMetadata(t *testing.T) {
	t.Run("base case", func(t *testing.T) {
		block := &Block{}
		blocks := []*Block{block}
		assert.Empty(t, block.ModifiedBy)
		assert.Empty(t, block.UpdateAt)

		StampModificationMetadata("user_id_1", blocks, nil)
		assert.Equal(t, "user_id_1", blocks[0].ModifiedBy)
		assert.NotEmpty(t, blocks[0].UpdateAt)
	})
}
func TestValidateBlockPatch(t *testing.T) {
	t.Run("Should return nil for block patch with valid updated fields", func(t *testing.T) {
		patch := &BlockPatch{
			ParentID: nil,
			Schema:   nil,
			Type:     nil,
			Title:    nil,
			UpdatedFields: map[string]interface{}{
				"field1": "value1",
				"field2": 123,
			},
			DeletedFields: nil,
		}

		err := ValidateBlockPatch(patch)

		require.NoError(t, err)
	})

	t.Run("Should return nil for block patch with valid file ID", func(t *testing.T) {
		patch := &BlockPatch{
			ParentID: nil,
			Schema:   nil,
			Type:     nil,
			Title:    nil,
			UpdatedFields: map[string]interface{}{
				"fileId": "xhwgf5r15fr3dryfozf1dmy41r9.jpg",
			},
			DeletedFields: nil,
		}

		err := ValidateBlockPatch(patch)

		require.NoError(t, err)
	})

	t.Run("Should return error for block patch with invalid file ID", func(t *testing.T) {
		patch := &BlockPatch{
			ParentID: nil,
			Schema:   nil,
			Type:     nil,
			Title:    nil,
			UpdatedFields: map[string]interface{}{
				"fileId": "../../../.../../././././././././filePath",
			},
			DeletedFields: nil,
		}

		err := ValidateBlockPatch(patch)

		require.Error(t, err)
		require.EqualError(t, err, "Invalid Block ID")
	})

	t.Run("Should return erro for blok patch with invalid attachment ID", func(t *testing.T) {
		patch := &BlockPatch{
			ParentID: nil,
			Schema:   nil,
			Type:     nil,
			Title:    nil,
			UpdatedFields: map[string]interface{}{
				"attchmentId": "../../../.../../././././././././filePath",
			},
			DeletedFields: nil,
		}

		err := ValidateBlockPatch(patch)

		require.Error(t, err)
		require.EqualError(t, err, "Invalid Block ID")
	})

	t.Run("Should return error for block patch with nested UpdatedFields", func(t *testing.T) {
		patch := &BlockPatch{
			ParentID: nil,
			Schema:   nil,
			Type:     nil,
			Title:    nil,
			UpdatedFields: map[string]interface{}{
				"0": "value1",
				"1": map[string]interface{}{
					"fileId": "../../../.../../././././././././filePath",
				},
			},
			DeletedFields: nil,
		}

		err := ValidateBlockPatch(patch)

		require.Error(t, err)
		require.EqualError(t, err, "Invalid Block ID")
	})
}

func TestValidateFileId(t *testing.T) {
	t.Run("Should return nil for valid file ID", func(t *testing.T) {
		fileID := "7xhwgf5r15fr3dryfozf1dmy41r.jpg"
		err := ValidateFileId(fileID)
		require.NoError(t, err)
	})

	t.Run("Should return error for empty file ID", func(t *testing.T) {
		fileID := ""
		err := ValidateFileId(fileID)
		require.Error(t, err)
		require.EqualError(t, err, "Block ID cannot be empty")
	})

	t.Run("Should return error for invalid file ID length", func(t *testing.T) {
		fileID := "7xhwgf5r15fr3dryfozf1dy1r9"
		err := ValidateFileId(fileID)
		require.Error(t, err)
		require.EqualError(t, err, "Invalid Block ID")
	})

	t.Run("Should return error for file ID with invalid characters", func(t *testing.T) {
		fileID := "../../../.../../././././././././filePath"
		err := ValidateFileId(fileID)
		require.Error(t, err)
		require.EqualError(t, err, "Invalid Block ID")
	})

	t.Run("Should return nil for valid legacy file ID", func(t *testing.T) {
		fileID := "729928ad8-4f83-1d9b-fdf7-d641616ae47b.jpg"
		err := ValidateFileId(fileID)
		require.NoError(t, err)
	})

	t.Run("Should return nil for valid legacy file ID without extension", func(t *testing.T) {
		fileID := "729928ad8-4f83-1d9b-fdf7-d641616ae47b"
		err := ValidateFileId(fileID)
		require.NoError(t, err)
	})

	t.Run("Should return nil for valid legacy file ID with arbitrary extension", func(t *testing.T) {
		fileID := "729928ad8-4f83-1d9b-fdf7-d641616ae47bEXTENSION"
		err := ValidateFileId(fileID)
		require.NoError(t, err)
	})

	t.Run("Should return error for invalid legacy file ID with non-hexadecimal characters", func(t *testing.T) {
		fileID := "7z9928ad8-4f83-1d9b-fdf7-d641616ae47b"
		err := ValidateFileId(fileID)
		require.EqualError(t, err, "Invalid Block ID")
	})
}

func TestBlockIsValid(t *testing.T) {
	t.Run("Should return nil for valid block", func(t *testing.T) {
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    string(utils.IDTypeNone) + mmModel.NewId(),
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      "Valid Block",
			Fields:     map[string]interface{}{},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValid()
		require.NoError(t, err)
	})

	t.Run("Should return error for block with empty BoardID", func(t *testing.T) {
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    "",
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      "Invalid Block",
			Fields:     map[string]interface{}{},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValid()
		require.Error(t, err)
		require.EqualError(t, err, "Block ID cannot be empty")
	})

	t.Run("Should return error for block with title exceeding max runes", func(t *testing.T) {
		longTitle := make([]rune, BlockTitleMaxRunes+1)
		for i := range longTitle {
			longTitle[i] = 'a'
		}
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    string(utils.IDTypeNone) + mmModel.NewId(),
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      string(longTitle),
			Fields:     map[string]interface{}{},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValid()
		require.Error(t, err)
		require.EqualError(t, err, "block title size limit exceeded")
	})

	t.Run("Should return error for block with fields exceeding max runes", func(t *testing.T) {
		longField := make([]rune, BlockFieldsMaxRunes+1)
		for i := range longField {
			longField[i] = 'a'
		}
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    string(utils.IDTypeNone) + mmModel.NewId(),
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      "Valid Block",
			Fields:     map[string]interface{}{"field": string(longField)},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValid()
		require.Error(t, err)
		require.EqualError(t, err, "block fields size limit exceeded")
	})

	t.Run("Should return error for block with invalid file ID in fields", func(t *testing.T) {
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    string(utils.IDTypeNone) + mmModel.NewId(),
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      "Valid Block",
			Fields:     map[string]interface{}{BlockFieldFileId: "invalid-file-id"},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValid()
		require.Error(t, err)
		require.EqualError(t, err, "Invalid Block ID")
	})

	t.Run("Should return error for block with invalid attachment ID in fields", func(t *testing.T) {
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    string(utils.IDTypeNone) + mmModel.NewId(),
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      "Valid Block",
			Fields:     map[string]interface{}{BlockFieldAttachmentId: "invalid-attachment-id"},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValid()
		require.Error(t, err)
		require.EqualError(t, err, "Invalid Block ID")
	})
}

func TestBlock_IsValidForImport(t *testing.T) {
	t.Run("Should return nil for valid block", func(t *testing.T) {
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    string(utils.IDTypeNone) + mmModel.NewId(),
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      "Valid Block",
			Fields:     map[string]interface{}{},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValidForImport()
		require.NoError(t, err)
	})

	t.Run("Should not return error for block with empty BoardID", func(t *testing.T) {
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    "",
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      "Invalid Block",
			Fields:     map[string]interface{}{},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValidForImport()
		require.NoError(t, err)
	})

	t.Run("Should return error for block with title exceeding max runes", func(t *testing.T) {
		longTitle := make([]rune, BlockTitleMaxRunes+1)
		for i := range longTitle {
			longTitle[i] = 'a'
		}
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    string(utils.IDTypeNone) + mmModel.NewId(),
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      string(longTitle),
			Fields:     map[string]interface{}{},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValidForImport()
		require.Error(t, err)
		require.EqualError(t, err, "block title size limit exceeded")
	})

	t.Run("Should return error for block with fields exceeding max runes", func(t *testing.T) {
		longField := make([]rune, BlockFieldsMaxRunes+1)
		for i := range longField {
			longField[i] = 'a'
		}
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    string(utils.IDTypeNone) + mmModel.NewId(),
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      "Valid Block",
			Fields:     map[string]interface{}{"field": string(longField)},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValidForImport()
		require.Error(t, err)
		require.EqualError(t, err, "block fields size limit exceeded")
	})

	t.Run("Should return error for block with invalid file ID in fields", func(t *testing.T) {
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    string(utils.IDTypeNone) + mmModel.NewId(),
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      "Valid Block",
			Fields:     map[string]interface{}{BlockFieldFileId: "invalid-file-id"},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValidForImport()
		require.Error(t, err)
		require.EqualError(t, err, "Invalid Block ID")
	})

	t.Run("Should return error for block with invalid attachment ID in fields", func(t *testing.T) {
		block := &Block{
			ID:         string(utils.IDTypeNone) + mmModel.NewId(),
			BoardID:    string(utils.IDTypeNone) + mmModel.NewId(),
			CreatedBy:  string(utils.IDTypeNone) + mmModel.NewId(),
			ModifiedBy: string(utils.IDTypeNone) + mmModel.NewId(),
			Schema:     1,
			Type:       TypeCard,
			Title:      "Valid Block",
			Fields:     map[string]interface{}{BlockFieldAttachmentId: "invalid-attachment-id"},
			CreateAt:   1234567890,
			UpdateAt:   1234567890,
		}
		err := block.IsValidForImport()
		require.Error(t, err)
		require.EqualError(t, err, "Invalid Block ID")
	})
}
