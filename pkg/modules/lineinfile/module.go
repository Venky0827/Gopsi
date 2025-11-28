package lineinfile

import (
    "context"
    "fmt"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "lineinfile" }

func (m mod) Validate(args map[string]any) error {
    if str(args["path"]) == "" || str(args["line"]) == "" { return fmt.Errorf("lineinfile requires path and line") }
    if str(args["state"]) == "" { args["state"] = "present" }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    path := str(args["path"])
    line := str(args["line"]) 
    re := str(args["regexp"]) 
    state := str(args["state"]) 
    var cmd string
    if re != "" { cmd = fmt.Sprintf("bash -lc 'grep -E %q %q >/dev/null 2>&1'", re, path) } else { cmd = fmt.Sprintf("bash -lc 'grep -F %q %q >/dev/null 2>&1'", line, path) }
    _, _, exit, err := c.Exec(ctx, cmd, nil, false)
    if err != nil { return module.Result{}, err }
    present := exit == 0
    if state == "present" { return module.Result{Changed: !present, Artifacts: map[string]any{"path": path, "present": present}}, nil }
    return module.Result{Changed: present, Artifacts: map[string]any{"path": path, "present": present}}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    path := str(args["path"]) 
    line := str(args["line"]) 
    re := str(args["regexp"]) 
    state := str(args["state"]) 
    sudo := boolVal(args["become"]) 
    var cmd string
    if state == "present" {
        if re != "" {
            cmd = fmt.Sprintf("bash -lc 'if grep -E %q %q >/dev/null 2>&1; then sed -i -E \"s/%s/%s/\" %q; else echo %q | sudo -n tee -a %q >/dev/null; fi'", re, path, re, line, path, line, path)
        } else {
            cmd = fmt.Sprintf("bash -lc 'grep -F %q %q >/dev/null 2>&1 || echo %q | sudo -n tee -a %q >/dev/null'", line, path, line, path)
        }
    } else {
        if re != "" {
            cmd = fmt.Sprintf("bash -lc 'sed -i -E \"\\:%s:d\" %q'", re, path)
        } else {
            cmd = fmt.Sprintf("bash -lc 'sed -i -e \"\\:%s:d\" %q'", line, path)
        }
    }
    out, errOut, exit, err := c.Exec(ctx, cmd, nil, sudo)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: exit == 0, Artifacts: map[string]any{"path": path, "stdout": out, "stderr": errOut, "exit": exit}}, nil
}

func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }
func boolVal(v any) bool { b, _ := v.(bool); return b }

func init() { module.Register(mod{}) }
