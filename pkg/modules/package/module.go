package packagex

import (
    "context"
    "fmt"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "package" }

func (m mod) Validate(args map[string]any) error {
    if str(args["name"]) == "" { return fmt.Errorf("package requires name") }
    if str(args["state"]) == "" { args["state"] = "present" }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"])
    state := str(args["state"]) 
    cmd := fmt.Sprintf("bash -lc 'if command -v dpkg >/dev/null; then dpkg -s %q >/dev/null 2>&1; elif command -v rpm >/dev/null; then rpm -q %q >/dev/null 2>&1; else exit 2; fi'", name, name)
    _, _, exit, err := c.Exec(ctx, cmd, nil, false)
    if err != nil { return module.Result{}, err }
    installed := exit == 0
    if state == "present" { return module.Result{Changed: !installed}, nil }
    return module.Result{Changed: installed}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"]) 
    state := str(args["state"]) 
    var cmd string
    if state == "present" {
        cmd = fmt.Sprintf("bash -lc 'if command -v apt-get >/dev/null; then sudo -n apt-get update -y && sudo -n apt-get install -y %q; elif command -v yum >/dev/null; then sudo -n yum install -y %q; fi'", name, name)
    } else {
        cmd = fmt.Sprintf("bash -lc 'if command -v apt-get >/dev/null; then sudo -n apt-get remove -y %q; elif command -v yum >/dev/null; then sudo -n yum remove -y %q; fi'", name, name)
    }
    _, _, _, err := c.Exec(ctx, cmd, nil, false)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: true}, nil
}

func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }

func init() { module.Register(mod{}) }
