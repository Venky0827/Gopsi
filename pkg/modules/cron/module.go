package cron

import (
    "context"
    "fmt"
    "strings"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "cron" }

func (m mod) Validate(args map[string]any) error {
    if str(args["name"]) == "" || str(args["job"]) == "" || str(args["user"]) == "" { return fmt.Errorf("cron requires name, job, user") }
    if str(args["state"]) == "" { args["state"] = "present" }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"]) 
    user := str(args["user"]) 
    cmd := fmt.Sprintf("bash -lc 'crontab -l -u %q 2>/dev/null | grep -F %q >/dev/null'", user, name)
    _, _, exit, err := c.Exec(ctx, cmd, nil, false)
    if err != nil { return module.Result{}, err }
    present := exit == 0
    state := str(args["state"]) 
    return module.Result{Changed: (state == "present" && !present) || (state == "absent" && present), Artifacts: map[string]any{"name": name, "user": user, "present": present}}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"])
    user := str(args["user"]) 
    job := str(args["job"]) 
    minute := str(args["minute"]) 
    hour := str(args["hour"]) 
    day := str(args["day"]) 
    month := str(args["month"]) 
    weekday := str(args["weekday"]) 
    if minute == "" { minute = "*" }
    if hour == "" { hour = "*" }
    if day == "" { day = "*" }
    if month == "" { month = "*" }
    if weekday == "" { weekday = "*" }
    line := fmt.Sprintf("%s %s %s %s %s %s # %s", minute, hour, day, month, weekday, job, name)
    add := fmt.Sprintf("bash -lc '(crontab -l -u %q 2>/dev/null; echo %q) | crontab -u %q -'", user, strings.ReplaceAll(line, "\"", "\\\""), user)
    del := fmt.Sprintf("bash -lc 'crontab -l -u %q 2>/dev/null | grep -v -F %q | crontab -u %q -'", user, name, user)
    state := str(args["state"]) 
    var cmd string
    if state == "present" { cmd = add } else { cmd = del }
    _, errOut, exit, err := c.Exec(ctx, cmd, nil, true)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: exit == 0, Artifacts: map[string]any{"user": user, "name": name, "exit": exit, "stderr": errOut}}, nil
}

func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }

func init() { module.Register(mod{}) }
