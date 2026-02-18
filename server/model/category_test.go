// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"fmt"
	"strings"
	"testing"
)

func TestCategory_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		category  Category
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid category",
			category: Category{
				ID:     "7abcdefghijklmnopqrstuvwxyz",
				Name:   "Valid Category",
				UserID: "user-id",
				TeamID: "team-id",
				Type:   CategoryTypeCustom,
			},
			expectErr: false,
		},
		{
			name: "Invalid ID",
			category: Category{
				ID:     "",
				Name:   "Valid Category",
				UserID: "user-id",
				TeamID: "team-id",
				Type:   CategoryTypeCustom,
			},
			expectErr: true,
			errMsg:    "ID cannot be empty",
		},
		{
			name: "Empty Name",
			category: Category{
				ID:     "7abcdefghijklmnopqrstuvwxyz",
				Name:   "  ",
				UserID: "user-id",
				TeamID: "team-id",
				Type:   CategoryTypeCustom,
			},
			expectErr: true,
			errMsg:    "category name cannot be empty",
		},
		{
			name: "Empty UserID",
			category: Category{
				ID:     "7abcdefghijklmnopqrstuvwxyz",
				Name:   "Valid Category",
				UserID: "  ",
				TeamID: "team-id",
				Type:   CategoryTypeCustom,
			},
			expectErr: true,
			errMsg:    "category user ID cannot be empty",
		},
		{
			name: "Empty TeamID",
			category: Category{
				ID:     "7abcdefghijklmnopqrstuvwxyz",
				Name:   "Valid Category",
				UserID: "user-id",
				TeamID: "  ",
				Type:   CategoryTypeCustom,
			},
			expectErr: true,
			errMsg:    "category team id ID cannot be empty",
		},
		{
			name: "Invalid Type",
			category: Category{
				ID:     "7abcdefghijklmnopqrstuvwxyz",
				Name:   "Valid Category",
				UserID: "user-id",
				TeamID: "team-id",
				Type:   "InvalidType",
			},
			expectErr: true,
			errMsg:    fmt.Sprintf("category type is invalid. Allowed types: %s and %s", CategoryTypeSystem, CategoryTypeCustom),
		},
		{
			name: "Invalid category ID length",
			category: Category{
				ID:     "abcedf",
				Name:   "Valid Category",
				UserID: "user-id",
				TeamID: "team-id",
				Type:   CategoryTypeCustom,
			},
			expectErr: true,
			errMsg:    "Invalid Block ID",
		},
		{
			name: "ID contains invalid characters",
			category: Category{
				ID:     "%*|@{?%{=).+$.\\_%&}-/#}>)]` ￼",
				Name:   "Valid Category",
				UserID: "user-id",
				TeamID: "team-id",
				Type:   CategoryTypeCustom,
			},
			expectErr: true,
			errMsg:    "Invalid Block ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.category.IsValid()
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
			if err != nil && tt.expectErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error message: %q, got: %q", tt.errMsg, err.Error())
			}
		})
	}
}
