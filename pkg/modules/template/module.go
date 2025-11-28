package template

import (
    "bytes"
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "os"
    "text/template"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "template" }

func (m mod) Validate(args map[string]any) error {
    if str(args["src"]) == "" || str(args["dest"]) == "" {
        return fmt.Errorf("template requires src and dest")
    }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    src := str(args["src"]) 
    dest := str(args["dest"]) 
    b, err := os.ReadFile(src)
    if err != nil { return module.Result{}, err }
    t, err := template.New("t").Option("missingkey=error").Parse(string(b))
    if err != nil { return module.Result{}, err }
    var buf bytes.Buffer
    vars := map[string]any{}
    if m, ok := args["vars"].(map[string]any); ok { vars = m }
    if err := t.Execute(&buf, vars); err != nil { return module.Result{}, err }
    sumNew := sum(buf.Bytes())
    rc, err := c.Get(ctx, dest)
    if err != nil { return module.Result{Changed: true}, nil }
    defer rc.Close()
    rb, _ := io.ReadAll(rc)
    sumOld := sum(rb)
    arts := map[string]any{"dest": dest, "before": sumOld, "after": sumNew}
    return module.Result{Changed: sumNew != sumOld, Data: map[string]any{"before": sumOld, "after": sumNew}, Artifacts: arts}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    src := str(args["src"]) 
    dest := str(args["dest"]) 
    mode := os.FileMode(0644)
    if v := str(args["mode"]); v != "" {
        if mv, err := parseOctal(v); err == nil { mode = mv }
    }
    b, err := os.ReadFile(src)
    if err != nil { return module.Result{}, err }
    t, err := template.New("t").Option("missingkey=error").Parse(string(b))
    if err != nil { return module.Result{}, err }
    var buf bytes.Buffer
    vars := map[string]any{}
    if m, ok := args["vars"].(map[string]any); ok { vars = m }
    if err := t.Execute(&buf, vars); err != nil { return module.Result{}, err }
    if err := c.Put(ctx, bytes.NewReader(buf.Bytes()), dest, mode); err != nil { return module.Result{}, err }
    arts := map[string]any{"dest": dest, "mode": fmt.Sprintf("%#o", mode)}
    return module.Result{Changed: true, Msg: "updated", Artifacts: arts}, nil
}

func sum(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }
func parseOctal(s string) (os.FileMode, error) { var m uint32; _, err := fmt.Sscanf(s, "%o", &m); return os.FileMode(m), err }

func init() { module.Register(mod{}) }
