package unarchive

import (
    "context"
    "fmt"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "unarchive" }

func (m mod) Validate(args map[string]any) error {
    if str(args["src"]) == "" || str(args["dest"]) == "" { return fmt.Errorf("unarchive requires src and dest") }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    dest := str(args["dest"]) 
    marker := dest + "/.gopsi_unarchive_marker"
    _, _, exit, err := c.Exec(ctx, fmt.Sprintf("test -e %q", marker), nil, false)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: exit != 0, Artifacts: map[string]any{"dest": dest}}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    src := str(args["src"]) 
    dest := str(args["dest"]) 
    sudo := boolVal(args["become"]) 
    var cmd string
    if hasSuffix(src, ".tar.gz") || hasSuffix(src, ".tgz") { cmd = fmt.Sprintf("bash -lc 'mkdir -p %q && tar -xzf %q -C %q && touch %q/.gopsi_unarchive_marker'", dest, src, dest, dest) }
    if cmd == "" && hasSuffix(src, ".zip") { cmd = fmt.Sprintf("bash -lc 'mkdir -p %q && unzip -o %q -d %q && touch %q/.gopsi_unarchive_marker'", dest, src, dest, dest) }
    if cmd == "" { cmd = fmt.Sprintf("bash -lc 'mkdir -p %q && tar -xf %q -C %q && touch %q/.gopsi_unarchive_marker'", dest, src, dest, dest) }
    _, errOut, exit, err := c.Exec(ctx, cmd, nil, sudo)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: exit == 0, Artifacts: map[string]any{"src": src, "dest": dest, "exit": exit, "stderr": errOut}}, nil
}

func hasSuffix(s, suf string) bool { return len(s) >= len(suf) && s[len(s)-len(suf):] == suf }
func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }
func boolVal(v any) bool { b, _ := v.(bool); return b }

func init() { module.Register(mod{}) }
