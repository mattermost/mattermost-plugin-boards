// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCardPropertyValues(t *testing.T) {
	testCases := []struct {
		name   string
		values map[string]any
		valid  bool
	}{
		{"string value", map[string]any{"prop-id": "82%"}, true},
		{"array of strings value", map[string]any{"prop-id": []any{"opt-1", "opt-2"}}, true},
		{"empty value", map[string]any{"prop-id": nil}, true},
		{"object value", map[string]any{"prop-id": map[string]any{}}, false},
		{"array with an object value", map[string]any{"prop-id": []any{"opt-1", map[string]any{}}}, false},
		{"number value", map[string]any{"prop-id": float64(82)}, false},
		{"boolean value", map[string]any{"prop-id": true}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardPropertyValues(tc.values)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestValidateCardPropertyTemplate(t *testing.T) {
	testCases := []struct {
		name     string
		template map[string]any
		valid    bool
	}{
		{
			"complete template",
			map[string]any{
				"id":      "prop-id",
				"name":    "Note",
				"type":    "text",
				"options": []any{map[string]any{"id": "opt-id", "value": "Done", "color": "propColorGreen"}},
			},
			true,
		},
		{"only an id", map[string]any{"id": "prop-id"}, true},
		{"missing id", map[string]any{"name": "Note"}, false},
		{"empty id", map[string]any{"id": ""}, false},
		{"object name", map[string]any{"id": "prop-id", "name": map[string]any{}}, false},
		{"object type", map[string]any{"id": "prop-id", "type": map[string]any{}}, false},
		{"object options", map[string]any{"id": "prop-id", "options": map[string]any{}}, false},
		{"option with an object value", map[string]any{"id": "prop-id", "options": []any{map[string]any{"id": "opt-id", "value": map[string]any{}}}}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardPropertyTemplate(tc.template)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestBoardPatchWithInvalidCardProperties(t *testing.T) {
	patch := &BoardPatch{
		UpdatedCardProperties: []map[string]any{
			{"id": "prop-id", "name": map[string]any{}, "type": "text", "options": []any{}},
		},
	}

	t.Run("is rejected by the patch validation", func(t *testing.T) {
		require.Error(t, patch.IsValid())
	})

	t.Run("is not merged into the board", func(t *testing.T) {
		board := &Board{
			CardProperties: []map[string]any{
				{"id": "prop-id", "name": "Note", "type": "text", "options": []any{}},
			},
		}

		patched := patch.Patch(board)
		require.Equal(t, "Note", patched.CardProperties[0]["name"])
	})
}

func TestCardPatchWithInvalidProperties(t *testing.T) {
	patch := &CardPatch{
		UpdatedProperties: map[string]any{"prop-id": map[string]any{}},
	}

	t.Run("is rejected by the patch validation", func(t *testing.T) {
		require.Error(t, patch.CheckValid())
	})

	t.Run("is not merged into the card", func(t *testing.T) {
		card := &Card{Properties: map[string]any{"prop-id": "82%"}}

		patched := patch.Patch(card)
		require.Equal(t, "82%", patched.Properties["prop-id"])
	})
}

func TestValidateBlockPatchProperties(t *testing.T) {
	t.Run("valid properties", func(t *testing.T) {
		patch := &BlockPatch{UpdatedFields: map[string]any{"properties": map[string]any{"prop-id": "82%"}}}
		require.NoError(t, ValidateBlockPatch(patch))
	})

	t.Run("object property value", func(t *testing.T) {
		patch := &BlockPatch{UpdatedFields: map[string]any{"properties": map[string]any{"prop-id": map[string]any{}}}}
		require.Error(t, ValidateBlockPatch(patch))
	})

	t.Run("properties is not an object", func(t *testing.T) {
		patch := &BlockPatch{UpdatedFields: map[string]any{"properties": "not-an-object"}}
		require.Error(t, ValidateBlockPatch(patch))
	})
}
