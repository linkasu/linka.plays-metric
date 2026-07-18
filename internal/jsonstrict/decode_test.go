package jsonstrict

import (
	"strings"
	"testing"
)

func TestDecodeObjectRejectsAmbiguousJSON(t *testing.T) {
	tests := map[string]string{
		"duplicate root":   `{"value":1,"value":2}`,
		"duplicate nested": `{"value":1,"nested":{"key":1,"key":2}}`,
		"unknown":          `{"value":1,"other":2}`,
		"trailing":         `{"value":1}{"value":2}`,
		"not object":       `[1]`,
	}
	type input struct {
		Value int `json:"value"`
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if err := DecodeObject([]byte(body), &input{}, DefaultMaxDepth); err == nil {
				t.Fatal("ambiguous JSON was accepted")
			}
		})
	}
}

func TestDecodeObjectRejectsExcessiveDepth(t *testing.T) {
	body := strings.Repeat(`{"v":`, 17) + `1` + strings.Repeat(`}`, 17)
	var destination map[string]any
	if err := DecodeObject([]byte(body), &destination, DefaultMaxDepth); err == nil {
		t.Fatal("excessive JSON depth was accepted")
	}
}

func TestDecodeObjectAcceptsTypedObject(t *testing.T) {
	var destination struct {
		Value int `json:"value"`
	}
	if err := DecodeObject([]byte(`{"value":1}`), &destination, DefaultMaxDepth); err != nil {
		t.Fatal(err)
	}
	if destination.Value != 1 {
		t.Fatalf("value = %d", destination.Value)
	}
}
