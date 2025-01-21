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
			errMsg:    "ID cannot be empty",
		},
		{
			name:      "Invalid ID length (too short)",
			id:        "A123",
			expectErr: true,
			errMsg:    "invalid ID",
		},
		{
			name:      "Invalid ID length (too long)",
			id:        "A12345678901234567890123456789",
			expectErr: true,
			errMsg:    "invalid ID",
		},
		{
			name:      "Invalid ID format (contains special characters)",
			id:        "A12345678901234567@#$%&*(!",
			expectErr: true,
			errMsg:    "invalid ID",
		},
		{
			name:      "Invalid ID format (empty middle section)",
			id:        "A                        Z", // Block type + invalid middle
			expectErr: true,
			errMsg:    "invalid ID",
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
