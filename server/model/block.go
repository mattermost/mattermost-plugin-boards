// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"unicode/utf8"

	"github.com/mattermost/mattermost-plugin-boards/server/services/audit"
	mmModel "github.com/mattermost/mattermost/server/public/model"
)

const (
	MinIdLength            = 27
	BlockTitleMaxBytes     = 65535                  // Maximum size of a TEXT column in MySQL
	BlockTitleMaxRunes     = BlockTitleMaxBytes / 4 // Assume a worst-case representation
	BlockFieldsMaxRunes    = 800000
	BlockFieldFileId       = "fileId"
	BlockFieldAttachmentId = "attachmentId"
)

var (
	ErrBlockEmptyBoardID            = errors.New("boardID is empty")
	ErrBlockTitleSizeLimitExceeded  = errors.New("block title size limit exceeded")
	ErrBlockFieldsSizeLimitExceeded = errors.New("block fields size limit exceeded")
)

// Block is the basic data unit
// swagger:model
type Block struct {
	// The id for this block
	// required: true
	ID string `json:"id"`

	// The id for this block's parent block. Empty for root blocks
	// required: false
	ParentID string `json:"parentId"`

	// The id for user who created this block
	// required: true
	CreatedBy string `json:"createdBy"`

	// The id for user who last modified this block
	// required: true
	ModifiedBy string `json:"modifiedBy"`

	// The schema version of this block
	// required: true
	Schema int64 `json:"schema"`

	// The block type
	// required: true
	Type BlockType `json:"type"`

	// The display title
	// required: false
	Title string `json:"title"`

	// The block fields
	// required: false
	Fields map[string]interface{} `json:"fields"`

	// The creation time in miliseconds since the current epoch
	// required: true
	CreateAt int64 `json:"createAt"`

	// The last modified time in miliseconds since the current epoch
	// required: true
	UpdateAt int64 `json:"updateAt"`

	// The deleted time in miliseconds since the current epoch. Set to indicate this block is deleted
	// required: false
	DeleteAt int64 `json:"deleteAt"`

	// Deprecated. The workspace id that the block belongs to
	// required: false
	WorkspaceID string `json:"-"`

	// The board id that the block belongs to
	// required: true
	BoardID string `json:"boardId"`

	// The sequential number of the card within the board (only for card blocks)
	// required: false
	Number int64 `json:"number,omitempty"`

	// The card code (board code + number, e.g., "AB-123") (only for card blocks)
	// required: false
	Code string `json:"code,omitempty"`

	// Indicates if the card is limited
	// required: false
	Limited bool `json:"limited,omitempty"`
}

// BlockPatch is a patch for modify blocks
// swagger:model
type BlockPatch struct {
	// The id for this block's parent block. Empty for root blocks
	// required: false
	ParentID *string `json:"parentId"`

	// The schema version of this block
	// required: false
	Schema *int64 `json:"schema"`

	// The block type
	// required: false
	Type *BlockType `json:"type"`

	// The display title
	// required: false
	Title *string `json:"title"`

	// The block updated fields
	// required: false
	UpdatedFields map[string]interface{} `json:"updatedFields"`

	// The block removed fields
	// required: false
	DeletedFields []string `json:"deletedFields"`
}

// BlockPatchBatch is a batch of IDs and patches for modify blocks
// swagger:model
type BlockPatchBatch struct {
	// The id's for of the blocks to patch
	BlockIDs []string `json:"block_ids"`

	// The BlockPatches to be applied
	BlockPatches []BlockPatch `json:"block_patches"`
}

// BoardModifier is a callback that can modify each board during an import.
// A cache of arbitrary data will be passed for each call and any changes
// to the cache will be preserved for the next call.
// Return true to import the block or false to skip import.
type BoardModifier func(board *Board, cache map[string]interface{}) bool

// BlockModifier is a callback that can modify each block during an import.
// A cache of arbitrary data will be passed for each call and any changes
// to the cache will be preserved for the next call.
// Return true to import the block or false to skip import.
type BlockModifier func(block *Block, cache map[string]interface{}) bool

func BlocksFromJSON(data io.Reader) []*Block {
	var blocks []*Block
	_ = json.NewDecoder(data).Decode(&blocks)
	return blocks
}

// IsValid checks the block for errors before inserting, and makes
// sure it complies with the requirements of a valid block.
func (b *Block) IsValid() error {
	if err := IsValidId(b.BoardID); err != nil {
		return err
	}

	return b.baseValidations()
}

func (b *Block) IsValidForImport() error {
	if err := IsValidId(b.BoardID); err != nil {
		if !errors.Is(err, errEmptyId) {
			return err
		}
	}

	return b.baseValidations()
}

func (b *Block) baseValidations() error {
	if utf8.RuneCountInString(b.Title) > BlockTitleMaxRunes {
		return ErrBlockTitleSizeLimitExceeded
	}

	fieldsJSON, err := json.Marshal(b.Fields)
	if err != nil {
		return err
	}

	if utf8.RuneCountInString(string(fieldsJSON)) > BlockFieldsMaxRunes {
		return ErrBlockFieldsSizeLimitExceeded
	}

	if fileID, ok := b.Fields[BlockFieldFileId].(string); ok {
		if err = ValidateFileId(fileID); err != nil {
			return err
		}
	}

	if attachmentId, ok := b.Fields[BlockFieldAttachmentId].(string); ok {
		if err = ValidateFileId(attachmentId); err != nil {
			return err
		}
	}

	return nil
}

var safeInputPattern = regexp.MustCompile(`^[a-zA-Z0-9 _-]+$`)

func ValidateFileId(id string) error {
	if id == "" {
		return errEmptyId
	}

	if len(id) < MinIdLength {
		return errInvalidId
	}

	//nolint:gocritic
	if mmModel.IsValidId(id[1:27]) {
		return nil
	} else if len(id) >= (legacyIDLength + 1) {
		return legacyIdCheck(id[1 : legacyIDLength+1])
	} else {
		return errInvalidId
	}
}

func ValidateBlockPatch(patch *BlockPatch) error {
	// Validate UpdatedFields map
	if patch.UpdatedFields != nil {
		if err := validateUpdatedFields(patch.UpdatedFields); err != nil {
			return err
		}
	}

	return nil
}

// validateUpdatedFields recursively checks keys and values for unsafe content.
func validateUpdatedFields(fields map[string]interface{}) error {
	for key, value := range fields {
		if !safeInputPattern.MatchString(key) {
			message := fmt.Sprintf("invalid characters in block with key: %s", key)
			return NewErrBadRequest(message)
		}

		if key == BlockFieldFileId {
			if strVal, ok := value.(string); ok {
				if err := ValidateFileId(strVal); err != nil {
					return err
				}
			}
		}

		if key == BlockFieldAttachmentId {
			if strVal, ok := value.(string); ok {
				if err := ValidateFileId(strVal); err != nil {
					return err
				}
			}
		}

		if nestedMap, ok := value.(map[string]interface{}); ok {
			if err := validateUpdatedFields(nestedMap); err != nil {
				return err
			}
		}
	}
	return nil
}

// LogClone implements the `mlog.LogCloner` interface to provide a subset of Block fields for logging.
func (b *Block) LogClone() interface{} {
	return struct {
		ID       string
		ParentID string
		BoardID  string
		Type     BlockType
	}{
		ID:       b.ID,
		ParentID: b.ParentID,
		BoardID:  b.BoardID,
		Type:     b.Type,
	}
}

// Patch returns an update version of the block.
func (p *BlockPatch) Patch(block *Block) *Block {
	if p.ParentID != nil {
		block.ParentID = *p.ParentID
	}

	if p.Schema != nil {
		block.Schema = *p.Schema
	}

	if p.Type != nil {
		block.Type = *p.Type
	}

	if p.Title != nil {
		block.Title = *p.Title
	}

	for key, field := range p.UpdatedFields {
		block.Fields[key] = field
	}

	for _, key := range p.DeletedFields {
		delete(block.Fields, key)
	}

	return block
}

type QueryBlocksOptions struct {
	BoardID   string    // if not empty then filter for blocks belonging to specified board
	ParentID  string    // if not empty then filter for blocks belonging to specified parent
	BlockType BlockType // if not empty and not `TypeUnknown` then filter for records of specified block type
	Page      int       // page number to select when paginating
	PerPage   int       // number of blocks per page (default=-1, meaning unlimited)
}

// QuerySubtreeOptions are query options that can be passed to GetSubTree methods.
type QuerySubtreeOptions struct {
	BeforeUpdateAt int64  // if non-zero then filter for records with update_at less than BeforeUpdateAt
	AfterUpdateAt  int64  // if non-zero then filter for records with update_at greater than AfterUpdateAt
	Limit          uint64 // if non-zero then limit the number of returned records
}

// QueryBlockHistoryOptions are query options that can be passed to GetBlockHistory.
type QueryBlockHistoryOptions struct {
	BeforeUpdateAt int64  // if non-zero then filter for records with update_at less than BeforeUpdateAt
	AfterUpdateAt  int64  // if non-zero then filter for records with update_at greater than AfterUpdateAt
	Limit          uint64 // if non-zero then limit the number of returned records
	Descending     bool   // if true then the records are sorted by insert_at in descending order
}

// QueryBoardHistoryOptions are query options that can be passed to GetBoardHistory.
type QueryBoardHistoryOptions struct {
	BeforeUpdateAt int64  // if non-zero then filter for records with update_at less than BeforeUpdateAt
	AfterUpdateAt  int64  // if non-zero then filter for records with update_at greater than AfterUpdateAt
	Limit          uint64 // if non-zero then limit the number of returned records
	Descending     bool   // if true then the records are sorted by insert_at in descending order
}

// QueryBlockHistoryOptions are query options that can be passed to GetBlockHistory.
type QueryBlockHistoryChildOptions struct {
	BeforeUpdateAt int64 // if non-zero then filter for records with update_at less than BeforeUpdateAt
	AfterUpdateAt  int64 // if non-zero then filter for records with update_at greater than AfterUpdateAt
	Page           int   // page number to select when paginating
	PerPage        int   // number of blocks per page (default=-1, meaning unlimited)
}

func StampModificationMetadata(userID string, blocks []*Block, auditRec *audit.Record) {
	now := GetMillis()
	for i := range blocks {
		blocks[i].ModifiedBy = userID
		blocks[i].UpdateAt = now

		if auditRec != nil {
			auditRec.AddMeta("block_"+strconv.FormatInt(int64(i), 10), blocks[i])
		}
	}
}

func (b *Block) ShouldBeLimited(cardLimitTimestamp int64) bool {
	return b.Type == TypeCard &&
		b.UpdateAt < cardLimitTimestamp
}

// Returns a limited version of the block that doesn't contain the
// contents of the block, only its IDs and type.
func (b *Block) GetLimited() *Block {
	newBlock := &Block{
		Title:       b.Title,
		ID:          b.ID,
		ParentID:    b.ParentID,
		BoardID:     b.BoardID,
		Schema:      b.Schema,
		Type:        b.Type,
		CreateAt:    b.CreateAt,
		UpdateAt:    b.UpdateAt,
		DeleteAt:    b.DeleteAt,
		WorkspaceID: b.WorkspaceID,
		Limited:     true,
	}

	if iconField, ok := b.Fields["icon"]; ok {
		newBlock.Fields = map[string]interface{}{
			"icon": iconField,
		}
	}

	return newBlock
}
