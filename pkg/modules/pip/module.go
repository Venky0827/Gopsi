package pip

import (
    "context"
    "fmt"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "pip" }

func (m mod) Validate(args map[string]any) error {
    if str(args["name"]) == "" { return fmt.Errorf("pip requires name") }
    if str(args["state"]) == "" { args["state"] = "present" }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"]) 
    venv := str(args["virtualenv"]) 
    cmd := fmt.Sprintf("bash -lc 'pip show %q >/dev/null 2>&1'", name)
    if venv != "" { cmd = fmt.Sprintf("bash -lc 'source %q/bin/activate && pip show %q >/dev/null 2>&1'", venv, name) }
    _, _, exit, err := c.Exec(ctx, cmd, nil, false)
    if err != nil { return module.Result{}, err }
    installed := exit == 0
    state := str(args["state"]) 
    changed := (state == "present" && !installed) || (state == "absent" && installed)
    return module.Result{Changed: changed, Artifacts: map[string]any{"name": name, "installed": installed, "venv": venv}}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"]) 
    state := str(args["state"]) 
    venv := str(args["virtualenv"]) 
    sudo := boolVal(args["become"]) 
    var cmd string
    if venv != "" {
        if state == "present" { cmd = fmt.Sprintf("bash -lc 'source %q/bin/activate && pip install %q'", venv, name) } else { cmd = fmt.Sprintf("bash -lc 'source %q/bin/activate && pip uninstall -y %q'", venv, name) }
    } else {
        if state == "present" { cmd = fmt.Sprintf("bash -lc 'pip install --user %q'", name) } else { cmd = fmt.Sprintf("bash -lc 'pip uninstall -y %q'", name) }
    }
    _, errOut, exit, err := c.Exec(ctx, cmd, nil, sudo)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: exit == 0, Artifacts: map[string]any{"name": name, "state": state, "venv": venv, "exit": exit, "stderr": errOut}}, nil
}

func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }
func boolVal(v any) bool { b, _ := v.(bool); return b }

func init() { module.Register(mod{}) }
