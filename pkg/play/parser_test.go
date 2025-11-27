package play

import (
    "os"
    "testing"
)

func TestLoadPlaybook(t *testing.T) {
    y := []byte(`- hosts: all
  vars: { a: 1 }
  tasks:
  - name: hello
    command: echo hello
`)
    f, err := os.CreateTemp(t.TempDir(), "pb-*.yml")
    if err != nil { t.Fatal(err) }
    if _, err := f.Write(y); err != nil { t.Fatal(err) }
    _ = f.Close()
    pb, err := LoadPlaybook(f.Name())
    if err != nil { t.Fatal(err) }
    if len(pb.Plays) != 1 { t.Fatalf("expected 1 play") }
    if pb.Plays[0].Hosts != "all" { t.Fatalf("wrong hosts") }
    if len(pb.Plays[0].Tasks) != 1 { t.Fatalf("expected 1 task") }
}
