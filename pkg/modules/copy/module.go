package copyx

import (
    "bytes"
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "text/template"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "copy" }

func (m mod) Validate(args map[string]any) error {
    dest := render(args, "dest")
    if dest == "" { return fmt.Errorf("copy requires dest") }
    if str(args["src"]) == "" && str(args["content"]) == "" { return fmt.Errorf("copy requires src or content") }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    dest := render(args, "dest")
    var data []byte
    if s := str(args["src"]); s != "" {
        b, err := os.ReadFile(s)
        if err != nil { return module.Result{}, err }
        data = b
    } else {
        data = []byte(str(args["content"]))
    }
    sumNew := sum(data)
    rc, err := c.Get(ctx, dest)
    if err != nil { return module.Result{Changed: true, Artifacts: map[string]any{"dest": dest, "after": sumNew}}, nil }
    defer rc.Close()
    rb, _ := io.ReadAll(rc)
    sumOld := sum(rb)
    return module.Result{Changed: sumNew != sumOld, Artifacts: map[string]any{"dest": dest, "before": sumOld, "after": sumNew}}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    dest := render(args, "dest")
    var data []byte
    if s := str(args["src"]); s != "" {
        b, err := os.ReadFile(s)
        if err != nil { return module.Result{}, err }
        data = b
    } else {
        data = []byte(str(args["content"]))
    }
    mode := os.FileMode(0644)
    if v := str(args["mode"]); v != "" { if mv, err := parseOctal(v); err == nil { mode = mv } }
    // ensure parent directory exists on remote
    dir := filepath.Dir(dest)
    _, _, _, _ = c.Exec(ctx, fmt.Sprintf("mkdir -p %q", dir), nil, true)
    if err := c.Put(ctx, bytes.NewReader(data), dest, mode); err != nil { return module.Result{}, err }
    return module.Result{Changed: true, Msg: "copied", Artifacts: map[string]any{"dest": dest}}, nil
}

func sum(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func parseOctal(s string) (os.FileMode, error) { var m uint32; _, err := fmt.Sscanf(s, "%o", &m); return os.FileMode(m), err }
func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }
func render(args map[string]any, key string) string {
    s := str(args[key])
    if s == "" { return s }
    vars, _ := args["vars"].(map[string]any)
    if vars == nil { return s }
    t, err := template.New("t").Option("missingkey=error").Parse(s)
    if err != nil { return s }
    var buf bytes.Buffer
    if err := t.Execute(&buf, vars); err != nil { return s }
    return buf.String()
}

func init() { module.Register(mod{}) }
