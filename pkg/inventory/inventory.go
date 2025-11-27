package inventory

import (
    "os"
    "path/filepath"

    "gopkg.in/yaml.v3"
)

type Host struct {
    Name string
    Addr string
    Vars map[string]any
}

type group struct {
    Children map[string]group       `yaml:"children"`
    Hosts    map[string]map[string]any `yaml:"hosts"`
    Vars     map[string]any         `yaml:"vars"`
}

type root struct {
    All group `yaml:"all"`
    SchemaVersion int `yaml:"schema_version"`
}

type Inventory struct {
    file string
    r    root
}

func LoadFromFile(path string) (*Inventory, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var r root
    if err := yaml.Unmarshal(b, &r); err != nil {
        return nil, err
    }
    if r.SchemaVersion == 0 { r.SchemaVersion = 1 }
    return &Inventory{file: path, r: r}, nil
}

func (i *Inventory) AllHosts(limit string) []Host {
    var out []Host
    var walk func(name string, g group)
    walk = func(name string, g group) {
        for h, vs := range g.Hosts {
            host := Host{Name: h, Vars: map[string]any{}}
            if v, ok := vs["host"]; ok {
                host.Addr, _ = v.(string)
            }
            for k, v := range g.Vars {
                host.Vars[k] = v
            }
            for k, v := range vs {
                host.Vars[k] = v
            }
            if limit == "" || matchLimit(limit, name, h) {
                out = append(out, host)
            }
        }
        for cn, cg := range g.Children {
            walk(cn, cg)
        }
    }
    walk("all", i.r.All)
    return out
}

func matchLimit(limit, groupName, hostName string) bool {
    if limit == "" {
        return true
    }
    if limit == groupName || limit == hostName {
        return true
    }
    return false
}

func (i *Inventory) BaseDir() string {
    return filepath.Dir(i.file)
}
