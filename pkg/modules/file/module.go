package file

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

func (m mod) Name() string { return "file" }

func (m mod) Validate(args map[string]any) error {
    path := render(args, "path")
    fname := render(args, "file_name")
    state := str(args["state"])
    if path == "" || fname == "" || state == "" { return fmt.Errorf("file requires path, file_name, state") }
    if state != "present" && state != "absent" { return fmt.Errorf("file state must be present or absent") }
    if state == "present" && str(args["content"]) == "" { return fmt.Errorf("file requires content when state=present") }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    base := render(args, "path")
    fname := render(args, "file_name")
    dest := filepath.Join(base, fname)
    state := str(args["state"])
    if state == "absent" {
        _, _, exit, err := c.Exec(ctx, fmt.Sprintf("test ! -e %q", dest), nil, false)
        if err != nil { return module.Result{}, err }
        return module.Result{Changed: exit != 0, Artifacts: map[string]any{"path": base, "file_name": fname, "dest": dest, "state": state, "exists": exit == 0}}, nil
    }
    rc, err := c.Get(ctx, dest)
    if err != nil { // not exists
        content := []byte(render(args, "content"))
        sumNew := sum(content)
        return module.Result{Changed: true, Artifacts: map[string]any{"path": base, "file_name": fname, "dest": dest, "before": "", "after": sumNew}}, nil
    }
    defer rc.Close()
    rb, _ := io.ReadAll(rc)
    sumOld := sum(rb)
    sumNew := sum([]byte(render(args, "content")))
    changed := sumNew != sumOld
    return module.Result{Changed: changed, Artifacts: map[string]any{"path": base, "file_name": fname, "dest": dest, "before": sumOld, "after": sumNew}}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    base := render(args, "path")
    fname := render(args, "file_name")
    dest := filepath.Join(base, fname)
    state := str(args["state"])
    if state == "absent" {
        _, _, _, err := c.Exec(ctx, fmt.Sprintf("rm -rf %q", dest), nil, true)
        if err != nil { return module.Result{}, err }
        return module.Result{Changed: true, Msg: "removed", Artifacts: map[string]any{"path": base, "file_name": fname, "dest": dest, "state": state}}, nil
    }
    _, _, _, err := c.Exec(ctx, fmt.Sprintf("mkdir -p %q", base), nil, true)
    if err != nil { return module.Result{}, err }
    mode := os.FileMode(0644)
    if v := str(args["mode"]); v != "" { if mv, err := parseOctal(v); err == nil { mode = mv } }
    data := []byte(render(args, "content"))
    if err := c.Put(ctx, bytes.NewReader(data), dest, mode); err != nil { return module.Result{}, err }
    return module.Result{Changed: true, Msg: "updated", Artifacts: map[string]any{"path": base, "file_name": fname, "dest": dest, "mode": fmt.Sprintf("%#o", mode)}}, nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
func render(args map[string]any, key string) string {
	s := str(args[key])
	if s == "" {
		return s
	}
	vars, _ := args["vars"].(map[string]any)
	if vars == nil {
		return s
	}
	t, err := template.New("t").Option("missingkey=error").Parse(s)
	if err != nil {
		return s
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return s
	}
	return buf.String()
}

func sum(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func parseOctal(s string) (os.FileMode, error) { var m uint32; _, err := fmt.Sscanf(s, "%o", &m); return os.FileMode(m), err }

func init() { module.Register(mod{}) }
