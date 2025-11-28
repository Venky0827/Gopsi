package runner

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"gopsi/pkg/conn"
	"gopsi/pkg/eval"
	"gopsi/pkg/facts"
	"gopsi/pkg/inventory"
	"gopsi/pkg/module"
	"gopsi/pkg/play"
)

type Runner struct {
	forks        int
	check        bool
	json         bool
	verbosity    int
	statsMu      sync.Mutex
	statsTotal   int
	statsSuccess int
	runStart     time.Time
}

func New(forks int, check bool) *Runner { return &Runner{forks: forks, check: check} }
func NewWithOptions(forks int, check bool, json bool, verbosity int) *Runner {
	return &Runner{forks: forks, check: check, json: json, verbosity: verbosity}
}

func (r *Runner) Run(ctx context.Context, hosts []inventory.Host, pb play.Playbook) error {
	if len(hosts) == 0 {
		return errors.New("no hosts to run")
	}
	if err := ensureModulesRegistered(); err != nil {
		return err
	}
	r.runStart = time.Now()
	var mu sync.Mutex
	var firstErr error
	for _, pl := range pb.Plays {
		var target []inventory.Host
		for _, h := range hosts {
			if pl.Hosts == "all" || pl.Hosts == h.Name {
				target = append(target, h)
			}
		}
		conc := r.forks
		if pl.Serial > 0 && pl.Serial < conc {
			conc = pl.Serial
		}
		sem := make(chan struct{}, conc)
		var wg sync.WaitGroup
		for _, h := range target {
			h := h
			sem <- struct{}{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				if err := r.runPlay(ctx, h, pl); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		if firstErr != nil {
			break
		}
	}
	if !r.json {
		dur := time.Since(r.runStart)
		r.verbosef(1, "")
		r.verbosef(1, summaryLine(r.statsSuccess, r.statsTotal, dur))
	}
	return firstErr
}

func (r *Runner) runPlay(ctx context.Context, h inventory.Host, pl play.Play) error {
	keyPath := expandHome(stringVar(h.Vars, "ssh_private_key_file"))
	if keyPath == "" {
		keyPath = filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		fallback := filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519")
		if _, statErr := os.Stat(fallback); statErr == nil {
			key, err = os.ReadFile(fallback)
		}
		if err != nil {
			return err
		}
	}
	user := stringVar(h.Vars, "user")
	if user == "" {
		user = os.Getenv("USER")
	}
	addr := h.Addr
	if addr == "" {
		addr = h.Name
	}
	r.verbosef(1, "HOST %s connect user=%s addr=%s key=%s", h.Name, user, addr, keyPath)
	c, err := conn.Dial(user, addr, key, 15*time.Second)
	if err != nil {
		return fmt.Errorf("%s: %w", h.Name, err)
	}
	defer c.Close()
	fs, err := facts.Gather(ctx, c)
	if err != nil {
		return err
	}
	r.verbosef(1, "%s facts %v", h.Name, fs)
	regs := map[string]any{"facts": map[string]any(fs)}
	vars := map[string]any{"facts": map[string]any(fs)}
	for k, v := range pl.Vars {
		vars[k] = v
	}
	for k, v := range h.Vars {
		vars[k] = v
	}
	for _, t := range pl.Tasks {
		if len(t.Tags) > 0 { /* tags filtering to be added */
		}
		m := module.Get(t.Module)
		if m == nil {
			return fmt.Errorf("unknown module: %s", t.Module)
		}
		// propagate become flag for modules that support it
		t.Args["become"] = pl.Become
		if err := m.Validate(t.Args); err != nil {
			r.verbosef(1, "%s validate error %s %v", h.Name, t.Name, err)
			return err
		}
		t.Args["vars"] = vars
		if t.When != "" {
			ok, err := eval.When(t.When, vars)
			if err != nil {
				r.verbosef(1, "%s when error %s %v", h.Name, t.Name, err)
				return err
			}
			if !ok {
				continue
			}
		}
		argsCopy := map[string]any{}
		for k, v := range t.Args {
			if k != "vars" {
				argsCopy[k] = v
			}
		}
		r.incTotal()
		if r.verbosity > 0 {
			r.verbosef(1, "")
		}
		r.verbosef(1, "TASK [%s] module=%s host=%s", t.Name, t.Module, h.Name)
		r.verbosef(2, "ARGS %s %v", t.Name, argsCopy)
		t0 := time.Now()
		res, err := m.Check(ctx, c, t.Args)
		if err != nil {
			r.verbosef(1, colorRed(fmt.Sprintf("%s check error %s %v", h.Name, t.Name, err)))
			return err
		}
		r.verbosef(2, "CHECK [%s] host=%s changed=%v msg=%s dur=%s", t.Name, h.Name, res.Changed, res.Msg, time.Since(t0))
		if r.verbosity >= 3 {
			r.verbosef(3, "DATA [%s] %s", t.Name, summarizeMap(res.Data, 512))
		}
		if r.check {
			r.printColored(h.Name, t.Name, res, true)
			r.incSuccess()
			continue
		}
		if res.Changed {
			t1 := time.Now()
			res, err = m.Apply(ctx, c, t.Args)
			if err != nil {
				r.verbosef(1, colorRed(fmt.Sprintf("%s apply error %s %v", h.Name, t.Name, err)))
				return err
			}
			r.verbosef(2, "APPLY [%s] host=%s changed=%v msg=%s dur=%s", t.Name, h.Name, res.Changed, res.Msg, time.Since(t1))
			if r.verbosity >= 3 {
				r.verbosef(3, "DATA [%s] %s", t.Name, summarizeMap(res.Data, 512))
				r.verbosef(3, "ARTIFACTS [%s] %s", t.Name, summarizeMap(res.Artifacts, 512))
			}
			r.printColored(h.Name, t.Name, res, false)
			r.incSuccess()
		} else {
			r.printColored(h.Name, t.Name, res, false)
			r.incSuccess()
		}
		if t.Register != "" {
			regs[t.Register] = res.Data
			if res.Artifacts != nil {
				regs[t.Register+"_artifacts"] = res.Artifacts
			}
		}
		if r.verbosity > 0 {
			r.verbosef(1, "")
		}
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

func (r *Runner) printColored(host, name string, res module.Result, check bool) {
	if r.json {
		r.print(host, name, res, check)
		return
	}
	line := fmt.Sprintf("%s | %s | changed=%v", host, name, res.Changed)
	if res.Changed {
		fmt.Println(colorYellow(line))
	} else {
		fmt.Println(colorGreen(line))
	}
}

func ensureModulesRegistered() error {
	// lazy registration
	// The actual registrations occur in init() of each module package when imported.
	// Import side-effects:
	var _ io.Reader
	_ = pem.Block{Type: "AT"}
	return nil
}

func (r *Runner) verbosef(level int, format string, a ...any) {
	if r.verbosity < level {
		return
	}
	if r.json {
		return
	}
	fmt.Printf(format+"\n", a...)
}

func stringVar(vars map[string]any, key string) string {
	v, ok := vars[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return os.ExpandEnv(s)
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		return filepath.Join(os.Getenv("HOME"), path[1:])
	}
	return path
}

func summarizeMap(m map[string]any, max int) string {
	if m == nil {
		return "{}"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := "{"
	for i, k := range keys {
		v := fmt.Sprintf("%v", m[k])
		if len(v) > max {
			v = v[:max] + "..."
		}
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%s=%s", k, v)
		if len(out) > max*2 {
			out += ", ..."
			break
		}
	}
	out += "}"
	return out
}

func (r *Runner) incTotal() {
	r.statsMu.Lock()
	r.statsTotal++
	r.statsMu.Unlock()
}

func (r *Runner) incSuccess() {
	r.statsMu.Lock()
	r.statsSuccess++
	r.statsMu.Unlock()
}

func summaryLine(success, total int, dur time.Duration) string {
	return fmt.Sprintf("SUMMARY: %s/%d tasks succeeded in %s", colorGreen(fmt.Sprintf("%d", success)), total, dur)
}

func colorGreen(s string) string  { return "\033[32m" + s + "\033[0m" }
func colorYellow(s string) string { return "\033[33m" + s + "\033[0m" }
func colorRed(s string) string    { return "\033[31m" + s + "\033[0m" }
