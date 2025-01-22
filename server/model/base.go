package model

import (
	mmModel "github.com/mattermost/mattermost/server/public/model"
)

const (
	IdLength = 27
)

var (
	errEmptyId   = NewErrBadRequest("Block ID cannot be empty")
	errInvalidId = NewErrBadRequest("Invalid Block ID")
)

func IsValidId(id string) error {
	if id == "" {
		return errEmptyId
	}

	// ID should have the right length
	if len(id) != IdLength {
		return errInvalidId
	}

	// ID should have the right format.
	// Excluding the first letter as it represents the block type
	// and is not part of the format validation
	if mmModel.IsValidId(id[1 : len(id)-1]) {
		return errInvalidId
	}

	return nil
}
