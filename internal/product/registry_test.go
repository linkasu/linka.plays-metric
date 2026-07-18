package product

import "testing"

func TestRegistryIsClosed(t *testing.T) {
	spec, ok := Lookup(LinkaPlays)
	if !ok || !spec.AllowsStream(StreamPlays) || !spec.AllowsGame("aquarium") {
		t.Fatal("LINKa Plays registry is incomplete")
	}
	if _, ok := Lookup("unknown"); ok || spec.AllowsStream("raw") || spec.AllowsGame("arbitrary-game") {
		t.Fatal("registry accepted an unknown compile-time value")
	}
}
