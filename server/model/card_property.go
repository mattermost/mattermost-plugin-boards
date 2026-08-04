// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package model

import "fmt"

type ErrInvalidCardProperty struct {
	msg string
}

func NewErrInvalidCardProperty(msg string) ErrInvalidCardProperty {
	return ErrInvalidCardProperty{msg: msg}
}

func (e ErrInvalidCardProperty) Error() string {
	return fmt.Sprintf("invalid card property, %s", e.msg)
}

// Card property templates and values are persisted as free-form JSON and
// rendered directly by the clients, which only handle strings and arrays of
// strings. Anything else is rejected before it reaches the database.
func IsValidCardPropertyValue(value any) bool {
	switch v := value.(type) {
	case nil, string, []string:
		return true
	case []any:
		for _, item := range v {
			if _, ok := item.(string); !ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func ValidateCardPropertyValues(values map[string]any) error {
	for propertyID, value := range values {
		if !IsValidCardPropertyValue(value) {
			return NewErrInvalidCardProperty(fmt.Sprintf("value of property %q must be a string or an array of strings", propertyID))
		}
	}

	return nil
}

func ValidateCardPropertyTemplate(template map[string]any) error {
	id, ok := template["id"].(string)
	if !ok || id == "" {
		return NewErrInvalidCardProperty("id must be a non empty string")
	}

	for _, field := range []string{"name", "type"} {
		value, exists := template[field]
		if !exists {
			continue
		}

		if _, ok := value.(string); !ok {
			return NewErrInvalidCardProperty(fmt.Sprintf("%s of property %q must be a string", field, id))
		}
	}

	return validateCardPropertyOptions(id, template["options"])
}

func ValidateCardPropertyTemplates(templates []map[string]any) error {
	for _, template := range templates {
		if err := ValidateCardPropertyTemplate(template); err != nil {
			return err
		}
	}

	return nil
}

func validateCardPropertyOptions(propertyID string, options any) error {
	invalidErr := NewErrInvalidCardProperty(fmt.Sprintf("options of property %q must be a list of objects with string fields", propertyID))

	switch opts := options.(type) {
	case nil:
		return nil
	case []any:
		for _, option := range opts {
			optionMap, ok := option.(map[string]any)
			if !ok || !isValidCardPropertyOption(optionMap) {
				return invalidErr
			}
		}
	case []map[string]any:
		for _, option := range opts {
			if !isValidCardPropertyOption(option) {
				return invalidErr
			}
		}
	default:
		return invalidErr
	}

	return nil
}

func isValidCardPropertyOption(option map[string]any) bool {
	for _, field := range []string{"id", "value", "color"} {
		value, exists := option[field]
		if !exists {
			continue
		}

		if _, ok := value.(string); !ok {
			return false
		}
	}

	return true
}
