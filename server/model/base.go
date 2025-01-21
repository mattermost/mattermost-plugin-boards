package model

import (
	"errors"

	mmModel "github.com/mattermost/mattermost/server/public/model"
)

const (
	IdLength = 27
)

func IsValidId(id string) error {
	if id == "" {
		return errors.New("ID cannot be empty")
	}

	// ID should have the right length
	if len(id) != IdLength {
		return errors.New("invalid ID")
	}

	// ID should have the right format.
	// Excluding the first letter as it represents the block type
	// and is not part of the format validation
	if mmModel.IsValidId(id[1 : len(id)-1]) {
		return errors.New("invalid ID")
	}

	return nil
}
