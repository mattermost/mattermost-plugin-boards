// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"github.com/mattermost/mattermost-plugin-boards/server/utils"
)

// RelationType represents the type of relationship between cards.
type RelationType string

const (
	RelationTypeBlocks         RelationType = "blocks"
	RelationTypeIsBlockedBy    RelationType = "is_blocked_by"
	RelationTypeRelatesTo      RelationType = "relates_to"
	RelationTypeDuplicates     RelationType = "duplicates"
	RelationTypeIsDuplicatedBy RelationType = "is_duplicated_by"
	RelationTypeClones         RelationType = "clones"
	RelationTypeIsClonedBy     RelationType = "is_cloned_by"
	RelationTypeCauses         RelationType = "causes"
	RelationTypeIsCausedBy     RelationType = "is_caused_by"
)

// GetInverseRelationType returns the inverse relation type for bidirectional relations.
func GetInverseRelationType(relationType RelationType) RelationType {
	switch relationType {
	case RelationTypeBlocks:
		return RelationTypeIsBlockedBy
	case RelationTypeIsBlockedBy:
		return RelationTypeBlocks
	case RelationTypeDuplicates:
		return RelationTypeIsDuplicatedBy
	case RelationTypeIsDuplicatedBy:
		return RelationTypeDuplicates
	case RelationTypeClones:
		return RelationTypeIsClonedBy
	case RelationTypeIsClonedBy:
		return RelationTypeClones
	case RelationTypeCauses:
		return RelationTypeIsCausedBy
	case RelationTypeIsCausedBy:
		return RelationTypeCauses
	case RelationTypeRelatesTo:
		return RelationTypeRelatesTo // symmetric relation
	default:
		return relationType
	}
}

// CardRelation represents a relationship between two cards
// swagger:model
type CardRelation struct {
	// The id for this relation
	// required: true
	ID string `json:"id"`

	// The id of the source card
	// required: true
	SourceCardID string `json:"sourceCardId"`

	// The id of the target card
	// required: true
	TargetCardID string `json:"targetCardId"`

	// The type of relation
	// required: true
	RelationType RelationType `json:"relationType"`

	// The id of the user who created this relation
	// required: true
	CreatedBy string `json:"createdBy"`

	// The creation time in milliseconds since the current epoch
	// required: true
	CreateAt int64 `json:"createAt"`

	// The id of the board (used for websocket broadcasts)
	BoardID string `json:"boardId,omitempty"`
}

// Populate populates a CardRelation with default values.
func (cr *CardRelation) Populate() {
	if cr.ID == "" {
		cr.ID = utils.NewID(utils.IDTypeNone)
	}
	if cr.CreateAt == 0 {
		cr.CreateAt = utils.GetMillis()
	}
}

// IsValid validates the card relation
func (cr *CardRelation) IsValid() error {
	if cr.ID == "" {
		return NewErrInvalidCardRelation("id cannot be empty")
	}
	if cr.SourceCardID == "" {
		return NewErrInvalidCardRelation("source card id cannot be empty")
	}
	if cr.TargetCardID == "" {
		return NewErrInvalidCardRelation("target card id cannot be empty")
	}
	if cr.SourceCardID == cr.TargetCardID {
		return NewErrInvalidCardRelation("source and target card cannot be the same")
	}
	if cr.RelationType == "" {
		return NewErrInvalidCardRelation("relation type cannot be empty")
	}
	if cr.CreatedBy == "" {
		return NewErrInvalidCardRelation("created by cannot be empty")
	}
	return nil
}

// ErrInvalidCardRelation is returned when a card relation is invalid
type ErrInvalidCardRelation struct {
	msg string
}

func (e ErrInvalidCardRelation) Error() string {
	return e.msg
}

func NewErrInvalidCardRelation(msg string) error {
	return ErrInvalidCardRelation{msg: msg}
}

// CardRelationWithCard represents a card relation with the related card details
// swagger:model
type CardRelationWithCard struct {
	CardRelation
	// The related card
	Card *Card `json:"card"`
}
