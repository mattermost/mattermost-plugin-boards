// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import "testing"

func TestIsValidId(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "Valid ID",
			id:        "A12345678901234567890123456", // Block type + 26 valid chars
			expectErr: false,
		},
		{
			name:      "Empty ID",
			id:        "",
			expectErr: true,
			errMsg:    "Block ID cannot be empty",
		},
		{
			name:      "Invalid ID length (too short)",
			id:        "A123",
			expectErr: true,
			errMsg:    "Invalid Block ID",
		},
		{
			name:      "Invalid ID length (too long)",
			id:        "A12345678901234567890123456789",
			expectErr: true,
			errMsg:    "Invalid Block ID",
		},
		{
			name:      "Invalid ID format (contains special characters)",
			id:        "A12345678901234567@#$%&*(!",
			expectErr: true,
			errMsg:    "Invalid Block ID",
		},
		{
			name:      "Invalid ID format (empty middle section)",
			id:        "A                        Z", // Block type + invalid middle
			expectErr: true,
			errMsg:    "Invalid Block ID",
		},
		{
			name:      "Valid legacy ID",
			id:        "c95fdfa9-4656-45d5-8865-4de6de626a72",
			expectErr: false,
		},
		{
			name:      "Invalid legacy ID",
			id:        "fdfa9-4656-45d5-8865-4de6de626a72",
			expectErr: true,
			errMsg:    "Invalid Block ID",
		},
		{
			name:      "Invalid legacy ID wth too few parts",
			id:        "fdfa9-4656-45d5-8865",
			expectErr: true,
			errMsg:    "Invalid Block ID",
		},
		{
			name:      "Invalid legacy ID wth non hexadecimal characters",
			id:        "zdfa9-4656-45d5-8865-4de6de626a72",
			expectErr: true,
			errMsg:    "Invalid Block ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsValidId(tt.id)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
			if err != nil && tt.expectErr && err.Error() != tt.errMsg {
				t.Errorf("expected error message: %q, got: %q", tt.errMsg, err.Error())
			}
		})
	}
}
