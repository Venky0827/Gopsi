package eval

import "testing"

func TestWhenEquals(t *testing.T) {
    ok, err := When("facts.os_family == \"Linux\"", map[string]any{"facts": map[string]any{"os_family": "Linux"}})
    if err != nil { t.Fatal(err) }
    if !ok { t.Fatalf("expected true") }
}

