package module

import (
    "context"
    "io"
    "os"
    "sort"
)

type Result struct {
    Changed bool
    Msg     string
    Data    map[string]any
    Artifacts map[string]any
}

type Module interface {
    Name() string
    Validate(args map[string]any) error
    Check(ctx context.Context, c Conn, args map[string]any) (Result, error)
    Apply(ctx context.Context, c Conn, args map[string]any) (Result, error)
}

type Conn interface {
    Exec(ctx context.Context, cmd string, env map[string]string, sudo bool) (string, string, int, error)
    Put(ctx context.Context, src io.Reader, dst string, mode os.FileMode) error
    Get(ctx context.Context, src string) (io.ReadCloser, error)
}

var registry = map[string]Module{}

func Register(m Module) { registry[m.Name()] = m }
func Get(name string) Module { return registry[name] }

func List() []string {
    names := make([]string, 0, len(registry))
    for n := range registry { names = append(names, n) }
    sort.Strings(names)
    return names
}
