package eval

import (
    "fmt"
    "strings"
)

func When(expr string, vars map[string]any) (bool, error) {
    e := strings.TrimSpace(expr)
    if e == "" { return true, nil }
    if strings.HasPrefix(e, "not ") {
        ok, err := When(strings.TrimSpace(strings.TrimPrefix(e, "not ")), vars)
        return !ok, err
    }
    parts := strings.SplitN(e, "==", 2)
    if len(parts) != 2 { return false, fmt.Errorf("unsupported expression: %s", expr) }
    left := strings.TrimSpace(parts[0])
    right := strings.TrimSpace(parts[1])
    right = strings.Trim(right, "\"'")
    val := get(vars, left)
    return fmt.Sprintf("%v", val) == right, nil
}

func get(m map[string]any, path string) any {
    cur := any(m)
    for _, p := range strings.Split(path, ".") {
        mm, ok := cur.(map[string]any)
        if !ok { return nil }
        cur = mm[p]
    }
    return cur
}
