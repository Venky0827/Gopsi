package geturl

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "get_url" }

func (m mod) Validate(args map[string]any) error {
    if str(args["url"]) == "" || str(args["dest"]) == "" { return fmt.Errorf("get_url requires url and dest") }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    dest := str(args["dest"]) 
    sumWant := str(args["checksum"]) 
    rc, err := c.Get(ctx, dest)
    if err != nil { return module.Result{Changed: true, Artifacts: map[string]any{"dest": dest}}, nil }
    defer rc.Close()
    b, _ := io.ReadAll(rc)
    sumHas := sum(b)
    if sumWant != "" {
        return module.Result{Changed: sumHas != sumWant, Artifacts: map[string]any{"dest": dest, "have": sumHas, "want": sumWant}}, nil
    }
    return module.Result{Changed: false, Artifacts: map[string]any{"dest": dest, "have": sumHas}}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    url := str(args["url"]) 
    dest := str(args["dest"]) 
    sudo := boolVal(args["become"]) 
    cmd := fmt.Sprintf("bash -lc 'if command -v curl >/dev/null; then curl -fsSL -o %q %q; else wget -q -O %q %q; fi'", dest, url, dest, url)
    _, errOut, exit, err := c.Exec(ctx, cmd, nil, sudo)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: exit == 0, Artifacts: map[string]any{"url": url, "dest": dest, "exit": exit, "stderr": errOut}}, nil
}

func sum(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }
func boolVal(v any) bool { b, _ := v.(bool); return b }

func init() { module.Register(mod{}) }
