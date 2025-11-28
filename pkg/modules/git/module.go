package gitx

import (
    "context"
    "fmt"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "git" }

func (m mod) Validate(args map[string]any) error {
    if str(args["repo"]) == "" || str(args["dest"]) == "" { return fmt.Errorf("git requires repo and dest") }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    dest := str(args["dest"]) 
    _, _, exit, err := c.Exec(ctx, fmt.Sprintf("test -e %q/.git", dest), nil, false)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: exit != 0, Artifacts: map[string]any{"dest": dest, "cloned": exit == 0}}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    repo := str(args["repo"]) 
    dest := str(args["dest"]) 
    version := str(args["version"]) 
    sudo := boolVal(args["become"]) 
    var cmd string
    cmd = fmt.Sprintf("bash -lc 'if [ -e %q/.git ]; then cd %q && git fetch --all && git checkout %q && git reset --hard origin/%s; else git clone %q %q && cd %q && git checkout %q; fi'", dest, dest, version, version, repo, dest, dest, version)
    if version == "" { cmd = fmt.Sprintf("bash -lc 'if [ -e %q/.git ]; then cd %q && git pull --ff-only; else git clone %q %q; fi'", dest, dest, repo, dest) }
    _, errOut, exit, err := c.Exec(ctx, cmd, nil, sudo)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: exit == 0, Artifacts: map[string]any{"repo": repo, "dest": dest, "version": version, "exit": exit, "stderr": errOut}}, nil
}

func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }
func boolVal(v any) bool { b, _ := v.(bool); return b }

func init() { module.Register(mod{}) }
