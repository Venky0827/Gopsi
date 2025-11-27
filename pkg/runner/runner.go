package runner

import (
    "context"
    "encoding/pem"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sync"
    "time"

    "gopsi/pkg/conn"
    "gopsi/pkg/inventory"
    "gopsi/pkg/module"
    "gopsi/pkg/play"
    "gopsi/pkg/facts"
    "gopsi/pkg/eval"
)

type Runner struct {
    forks int
    check bool
    json  bool
}

func New(forks int, check bool) *Runner { return &Runner{forks: forks, check: check} }
func NewWithOptions(forks int, check bool, json bool) *Runner { return &Runner{forks: forks, check: check, json: json} }

func (r *Runner) Run(ctx context.Context, hosts []inventory.Host, pb play.Playbook) error {
    if len(hosts) == 0 { return errors.New("no hosts to run") }
    if err := ensureModulesRegistered(); err != nil { return err }
    var mu sync.Mutex
    var firstErr error
    for _, pl := range pb.Plays {
        var target []inventory.Host
        for _, h := range hosts {
            if pl.Hosts == "all" || pl.Hosts == h.Name { target = append(target, h) }
        }
        conc := r.forks
        if pl.Serial > 0 && pl.Serial < conc { conc = pl.Serial }
        sem := make(chan struct{}, conc)
        var wg sync.WaitGroup
        for _, h := range target {
            h := h
            sem <- struct{}{}
            wg.Add(1)
            go func() {
                defer wg.Done()
                defer func(){ <-sem }()
                if err := r.runPlay(ctx, h, pl); err != nil {
                    mu.Lock()
                    if firstErr == nil { firstErr = err }
                    mu.Unlock()
                }
            }()
        }
        wg.Wait()
        if firstErr != nil { break }
    }
    return firstErr
}

func (r *Runner) runPlay(ctx context.Context, h inventory.Host, pl play.Play) error {
    keyPath := expandHome(stringVar(h.Vars, "ssh_private_key_file"))
    if keyPath == "" { keyPath = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa") }
    key, err := os.ReadFile(keyPath)
    if err != nil {
        fallback := filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519")
        if _, statErr := os.Stat(fallback); statErr == nil {
            key, err = os.ReadFile(fallback)
        }
        if err != nil { return err }
    }
    user := stringVar(h.Vars, "user")
    if user == "" { user = os.Getenv("USER") }
    addr := h.Addr
    if addr == "" { addr = h.Name }
    c, err := conn.Dial(user, addr, key, 15*time.Second)
    if err != nil { return fmt.Errorf("%s: %w", h.Name, err) }
    defer c.Close()
    fs, err := facts.Gather(ctx, c)
    if err != nil { return err }
    regs := map[string]any{"facts": map[string]any(fs)}
    vars := map[string]any{"facts": map[string]any(fs)}
    for k, v := range pl.Vars { vars[k] = v }
    for k, v := range h.Vars { vars[k] = v }
    for _, t := range pl.Tasks {
        if len(t.Tags) > 0 { /* tags filtering to be added */ }
        m := module.Get(t.Module)
        if m == nil { return fmt.Errorf("unknown module: %s", t.Module) }
        if err := m.Validate(t.Args); err != nil { return err }
        t.Args["vars"] = vars
        if t.When != "" {
            ok, err := eval.When(t.When, vars)
            if err != nil { return err }
            if !ok { continue }
        }
        res, err := m.Check(ctx, c, t.Args)
        if err != nil { return err }
        if r.check {
            r.print(h.Name, t.Name, res, true)
            continue
        }
        if res.Changed {
            res, err = m.Apply(ctx, c, t.Args)
            if err != nil { return err }
            r.print(h.Name, t.Name, res, false)
        } else {
            r.print(h.Name, t.Name, res, false)
        }
        if t.Register != "" { regs[t.Register] = res.Data }
    }
    return nil
}

func (r *Runner) print(host, name string, res module.Result, check bool) {
    if r.json {
        fmt.Printf("{\"host\":%q,\"task\":%q,\"changed\":%v,\"check\":%v,\"msg\":%q}\n", host, name, res.Changed, check, res.Msg)
        return
    }
    fmt.Printf("%s | %s | changed=%v\n", host, name, res.Changed)
}

func ensureModulesRegistered() error {
    // lazy registration
    // The actual registrations occur in init() of each module package when imported.
    // Import side-effects:
    var _ io.Reader
    _ = pem.Block{Type: "AT"}
    return nil
}

func stringVar(vars map[string]any, key string) string {
    v, ok := vars[key]
    if !ok || v == nil { return "" }
    s, ok := v.(string)
    if !ok { return "" }
    return os.ExpandEnv(s)
}

func expandHome(path string) string {
    if path == "" { return path }
    if path[0] == '~' {
        return filepath.Join(os.Getenv("HOME"), path[1:])
    }
    return path
}
