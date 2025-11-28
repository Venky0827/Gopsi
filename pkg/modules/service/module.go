package service

import (
    "context"
    "fmt"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "service" }

func (m mod) Validate(args map[string]any) error {
    if str(args["name"]) == "" { return fmt.Errorf("service requires name") }
    if str(args["state"]) == "" { args["state"] = "started" }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"]) 
    state := str(args["state"]) 
    cmd := fmt.Sprintf("systemctl is-active %q", name)
    out, _, _, err := c.Exec(ctx, cmd, nil, false)
    if err != nil { return module.Result{}, err }
    active := out == "active\n"
    switch state {
    case "started":
        arts := map[string]any{"name": name, "state": state, "active": active}
        return module.Result{Changed: !active, Artifacts: arts}, nil
    case "stopped":
        arts := map[string]any{"name": name, "state": state, "active": active}
        return module.Result{Changed: active, Artifacts: arts}, nil
    case "restarted":
        arts := map[string]any{"name": name, "state": state}
        return module.Result{Changed: true, Artifacts: arts}, nil
    default:
        return module.Result{Changed: false, Artifacts: map[string]any{"name": name, "state": state, "active": active}}, nil
    }
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"]) 
    state := str(args["state"]) 
    var cmd string
    switch state {
    case "started":
        cmd = fmt.Sprintf("sudo -n systemctl start %q", name)
    case "stopped":
        cmd = fmt.Sprintf("sudo -n systemctl stop %q", name)
    case "restarted":
        cmd = fmt.Sprintf("sudo -n systemctl restart %q", name)
    default:
        return module.Result{Changed: false}, nil
    }
    _, _, _, err := c.Exec(ctx, cmd, nil, false)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: true, Artifacts: map[string]any{"name": name, "state": state, "cmd": cmd}}, nil
}

func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }

func init() { module.Register(mod{}) }
