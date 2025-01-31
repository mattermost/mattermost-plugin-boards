// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"regexp"
	"strings"

	mmModel "github.com/mattermost/mattermost/server/public/model"
)

const (
	IdLength       = 27
	legacyIDLength = 36
)

var (
	errEmptyId   = NewErrBadRequest("Block ID cannot be empty")
	errInvalidId = NewErrBadRequest("Invalid Block ID")
)

func IsValidId(id string) error {
	if id == "" {
		return errEmptyId
	}

	switch len(id) {
	case IdLength:
		return newIdCheck(id)
	case legacyIDLength:
		return legacyIdCheck(id)
	default:
		return errInvalidId
	}
}

func newIdCheck(id string) error {
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

func legacyIdCheck(id string) error {
	// Check if the ID is empty
	if id == "" {
		return errInvalidId
	}

	// Check total length (exact 36 characters including hyphens)
	if len(id) != 36 {
		return errInvalidId
	}

	// Check hyphen positions
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return errInvalidId
	}

	// Split the ID into segments based on hyphens
	segments := strings.Split(id, "-")

	// Validate each segment's length
	if len(segments[0]) != 8 ||
		len(segments[1]) != 4 ||
		len(segments[2]) != 4 ||
		len(segments[3]) != 4 ||
		len(segments[4]) != 12 {
		return errInvalidId
	}

	// Remove hyphens for character validation
	cleanID := strings.ReplaceAll(id, "-", "")

	// Regex to ensure only hexadecimal characters are used
	match, _ := regexp.MatchString("^[0-9a-f]+$", cleanID)

	if !match {
		return errInvalidId
	}

	return nil
}
