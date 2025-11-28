package command

import (
    "context"
    "fmt"
    "strings"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "command" }

func (m mod) Validate(args map[string]any) error {
    if _, ok := args["_"].(string); !ok {
        return fmt.Errorf("command requires a string value")
    }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    creates := str(args["creates"]) 
    removes := str(args["removes"]) 
    if creates != "" {
        _, _, exit, err := c.Exec(ctx, fmt.Sprintf("test -e %q", creates), nil, false)
        if err != nil { return module.Result{}, err }
        if exit == 0 { return module.Result{Changed: false, Msg: "exists"}, nil }
        return module.Result{Changed: true}, nil
    }
    if removes != "" {
        _, _, exit, err := c.Exec(ctx, fmt.Sprintf("test ! -e %q", removes), nil, false)
        if err != nil { return module.Result{}, err }
        if exit == 0 { return module.Result{Changed: false, Msg: "absent"}, nil }
        return module.Result{Changed: true}, nil
    }
    return module.Result{Changed: true}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    cmd := str(args["_"]) 
    env := map[string]string{}
    if e, ok := args["env"].(map[string]any); ok {
        for k, v := range e { env[k] = fmt.Sprintf("%v", v) }
    }
    sudo := boolVal(args["become"]) 
    out, errOut, exit, err := c.Exec(ctx, cmd, env, sudo)
    if err != nil { return module.Result{}, err }
    msg := strings.TrimSpace(out + "\n" + errOut)
    arts := map[string]any{"stdout": out, "stderr": errOut, "exit": exit, "cmd": cmd, "sudo": sudo}
    return module.Result{Changed: exit == 0, Msg: msg, Artifacts: arts}, nil
}

func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }
func boolVal(v any) bool { b, _ := v.(bool); return b }

func init() { module.Register(mod{}) }
