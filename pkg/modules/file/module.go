package file

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "file" }

func (m mod) Validate(args map[string]any) error {
	if render(args, "path") == "" {
		return fmt.Errorf("file requires path")
	}
	return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
	p := render(args, "path")
	state := str(args["state"])
	if state == "absent" {
		_, _, exit, err := c.Exec(ctx, fmt.Sprintf("test ! -e %q", p), nil, false)
		if err != nil {
			return module.Result{}, err
		}
		return module.Result{Changed: exit != 0, Artifacts: map[string]any{"path": p, "state": state, "exists": exit == 0}}, nil
	}
	_, _, exit, err := c.Exec(ctx, fmt.Sprintf("test -e %q", p), nil, false)
	if err != nil {
		return module.Result{}, err
	}
	return module.Result{Changed: exit != 0, Artifacts: map[string]any{"path": p, "state": state, "exists": exit == 0}}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
	p := render(args, "path")
	state := str(args["state"])
	if state == "absent" {
		_, _, _, err := c.Exec(ctx, fmt.Sprintf("rm -rf %q", p), nil, true)
		if err != nil {
			return module.Result{}, err
		}
		return module.Result{Changed: true, Msg: "removed", Artifacts: map[string]any{"path": p, "state": state}}, nil
	}
	_, _, _, err := c.Exec(ctx, fmt.Sprintf("mkdir -p $(dirname %q) && touch %q", p, p), nil, true)
	if err != nil {
		return module.Result{}, err
	}
	return module.Result{Changed: true, Msg: "created", Artifacts: map[string]any{"path": p, "state": state}}, nil
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

func init() { module.Register(mod{}) }
