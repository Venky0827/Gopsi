package play

import (
    "os"

    "gopkg.in/yaml.v3"
)

func LoadPlaybook(path string) (Playbook, error) {
    b, err := os.ReadFile(path)
    if err != nil { return Playbook{}, err }
    var pb Playbook
    var rawList []map[string]any
    if err := yaml.Unmarshal(b, &rawList); err == nil && len(rawList) > 0 {
        for _, p := range rawList { parsePlay(&pb, p) }
        if pb.SchemaVersion == 0 { pb.SchemaVersion = 1 }
        return pb, nil
    }
    var rawMap map[string]any
    if err := yaml.Unmarshal(b, &rawMap); err != nil { return Playbook{}, err }
    if sv, ok := rawMap["schema_version"].(int); ok { pb.SchemaVersion = sv } else { pb.SchemaVersion = 1 }
    if ps, ok := rawMap["plays"].([]any); ok {
        for _, p := range ps { parsePlay(&pb, p.(map[string]any)) }
    }
    return pb, nil
}

func parsePlay(pb *Playbook, p map[string]any) {
    var pl Play
    if v, ok := p["hosts"].(string); ok { pl.Hosts = v }
    if v, ok := p["become"].(bool); ok { pl.Become = v }
    if v, ok := p["vars"].(map[string]any); ok { pl.Vars = v }
    if v, ok := p["serial"].(int); ok { pl.Serial = v }
    if ts, ok := p["tasks"].([]any); ok {
        for _, t := range ts {
            tm, _ := t.(map[string]any)
            task := Task{Raw: tm}
            if v, ok := tm["name"].(string); ok { task.Name = v }
            if v, ok := tm["tags"].([]any); ok { for _, x := range v { if s, ok := x.(string); ok { task.Tags = append(task.Tags, s) } } }
            if v, ok := tm["when"].(string); ok { task.When = v }
            if v, ok := tm["notify"].([]any); ok { for _, x := range v { if s, ok := x.(string); ok { task.Notify = append(task.Notify, s) } } }
            if v, ok := tm["register"].(string); ok { task.Register = v }
            for k, val := range tm {
                switch k {
                case "name", "tags", "when", "notify", "register":
                default:
                    task.Module = k
                    if args, ok := val.(map[string]any); ok { task.Args = args } else { task.Args = map[string]any{"_": val} }
                }
            }
            pl.Tasks = append(pl.Tasks, task)
        }
    }
    if hs, ok := p["handlers"].([]any); ok {
        for _, t := range hs {
            tm, _ := t.(map[string]any)
            task := Task{Raw: tm}
            if v, ok := tm["name"].(string); ok { task.Name = v }
            for k, val := range tm { if k == "name" { continue }; task.Module = k; if args, ok := val.(map[string]any); ok { task.Args = args } else { task.Args = map[string]any{"_": val} } }
            pl.Handlers = append(pl.Handlers, task)
        }
    }
    pb.Plays = append(pb.Plays, pl)
}
