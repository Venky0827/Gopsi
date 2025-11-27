package file

import (
    "context"
    "fmt"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "file" }

func (m mod) Validate(args map[string]any) error {
    if str(args["path"]) == "" {
        return fmt.Errorf("file requires path")
    }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    p := str(args["path"]) 
    state := str(args["state"]) 
    if state == "absent" {
        _, _, exit, err := c.Exec(ctx, fmt.Sprintf("test ! -e %q", p), nil, false)
        if err != nil { return module.Result{}, err }
        return module.Result{Changed: exit != 0}, nil
    }
    _, _, exit, err := c.Exec(ctx, fmt.Sprintf("test -e %q", p), nil, false)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: exit != 0}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    p := str(args["path"]) 
    state := str(args["state"]) 
    if state == "absent" {
        _, _, _, err := c.Exec(ctx, fmt.Sprintf("rm -rf %q", p), nil, true)
        if err != nil { return module.Result{}, err }
        return module.Result{Changed: true, Msg: "removed"}, nil
    }
    _, _, _, err := c.Exec(ctx, fmt.Sprintf("mkdir -p $(dirname %q) && touch %q", p, p), nil, true)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: true, Msg: "created"}, nil
}

func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }

func init() { module.Register(mod{}) }
