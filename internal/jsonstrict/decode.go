package jsonstrict

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const DefaultMaxDepth = 16

// DecodeObject rejects ambiguous JSON before decoding it into a typed contract.
func DecodeObject(data []byte, destination any, maxDepth int) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return errors.New("expected JSON object")
	}
	if maxDepth < 1 {
		return errors.New("max depth must be positive")
	}
	if err := validate(trimmed, maxDepth); err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("unexpected data after JSON object")
	}
	return nil
}

func validate(data []byte, maxDepth int) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := validateValue(decoder, 1, maxDepth); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected data after JSON value")
		}
		return fmt.Errorf("invalid trailing JSON: %w", err)
	}
	return nil
}

func validateValue(decoder *json.Decoder, depth, maxDepth int) error {
	if depth > maxDepth {
		return fmt.Errorf("JSON nesting exceeds maximum depth %d", maxDepth)
	}
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}

	switch delimiter {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("invalid object key: %w", err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if _, exists := keys[key]; exists {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			keys[key] = struct{}{}
			if err := validateValue(decoder, depth+1, maxDepth); err != nil {
				return err
			}
		}
		return expectDelimiter(decoder, '}')
	case '[':
		for decoder.More() {
			if err := validateValue(decoder, depth+1, maxDepth); err != nil {
				return err
			}
		}
		return expectDelimiter(decoder, ']')
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

func expectDelimiter(decoder *json.Decoder, expected json.Delim) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON container: %w", err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != expected {
		return fmt.Errorf("expected JSON delimiter %q", expected)
	}
	return nil
}
