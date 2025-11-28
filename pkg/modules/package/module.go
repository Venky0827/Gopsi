package packagex

import (
    "context"
    "fmt"

    "gopsi/pkg/module"
)

type mod struct{}

func (m mod) Name() string { return "package" }

func (m mod) Validate(args map[string]any) error {
    if str(args["name"]) == "" { return fmt.Errorf("package requires name") }
    act := str(args["action"])
    if act == "" {
        // backward compat mapping from state
        st := str(args["state"]) 
        if st == "" { act = "install" } else if st == "present" { act = "install" } else if st == "absent" { act = "remove" } else { act = st }
        args["action"] = act
    }
    if act != "install" && act != "remove" { return fmt.Errorf("package action must be install or remove") }
    return nil
}

func (m mod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"]) 
    act := str(args["action"]) 
    mgr := detectManager(ctx, c)
    var chk string
    switch mgr {
    case "apt":
        chk = fmt.Sprintf("bash -lc 'dpkg -s %q >/dev/null 2>&1'", name)
    case "dnf", "yum", "rpm":
        chk = fmt.Sprintf("bash -lc 'rpm -q %q >/dev/null 2>&1'", name)
    case "apk":
        chk = fmt.Sprintf("bash -lc 'apk info -e %q >/dev/null 2>&1'", name)
    case "zypper":
        chk = fmt.Sprintf("bash -lc 'rpm -q %q >/dev/null 2>&1'", name)
    default:
        chk = fmt.Sprintf("bash -lc 'rpm -q %q >/dev/null 2>&1'", name)
    }
    _, _, exit, err := c.Exec(ctx, chk, nil, false)
    if err != nil { return module.Result{}, err }
    installed := exit == 0
    changed := false
    if act == "install" { changed = !installed } else { changed = installed }
    return module.Result{Changed: changed, Artifacts: map[string]any{"name": name, "manager": mgr, "installed": installed}}, nil
}

func (m mod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"]) 
    act := str(args["action"]) 
    mgr := detectManager(ctx, c)
    var cmd string
    switch mgr {
    case "apt":
        if act == "install" { cmd = fmt.Sprintf("bash -lc 'sudo -n apt-get update -y && sudo -n apt-get install -y %q'", name) } else { cmd = fmt.Sprintf("bash -lc 'sudo -n apt-get remove -y %q'", name) }
    case "dnf":
        if act == "install" { cmd = fmt.Sprintf("bash -lc 'sudo -n dnf install -y %q'", name) } else { cmd = fmt.Sprintf("bash -lc 'sudo -n dnf remove -y %q'", name) }
    case "yum":
        if act == "install" { cmd = fmt.Sprintf("bash -lc 'sudo -n yum install -y %q'", name) } else { cmd = fmt.Sprintf("bash -lc 'sudo -n yum remove -y %q'", name) }
    case "apk":
        if act == "install" { cmd = fmt.Sprintf("bash -lc 'sudo -n apk add %q'", name) } else { cmd = fmt.Sprintf("bash -lc 'sudo -n apk del %q'", name) }
    case "zypper":
        if act == "install" { cmd = fmt.Sprintf("bash -lc 'sudo -n zypper -n install -y %q'", name) } else { cmd = fmt.Sprintf("bash -lc 'sudo -n zypper -n remove -y %q'", name) }
    default:
        if act == "install" { cmd = fmt.Sprintf("bash -lc 'sudo -n rpm -Uvh %q'", name) } else { cmd = fmt.Sprintf("bash -lc 'sudo -n rpm -e %q'", name) }
    }
    _, _, exit, err := c.Exec(ctx, cmd, nil, false)
    if err != nil { return module.Result{}, err }
    return module.Result{Changed: true, Artifacts: map[string]any{"name": name, "action": act, "manager": mgr, "cmd": cmd, "exit": exit}}, nil
}

func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }

func init() { module.Register(mod{}) }

func detectManager(ctx context.Context, c module.Conn) string {
    // Prefer RHEL derivatives first: dnf, yum; then apk, zypper, apt; fallback rpm
    if _, _, exit, _ := c.Exec(ctx, "command -v dnf >/dev/null", nil, false); exit == 0 { return "dnf" }
    if _, _, exit, _ := c.Exec(ctx, "command -v yum >/dev/null", nil, false); exit == 0 { return "yum" }
    if _, _, exit, _ := c.Exec(ctx, "command -v apk >/dev/null", nil, false); exit == 0 { return "apk" }
    if _, _, exit, _ := c.Exec(ctx, "command -v zypper >/dev/null", nil, false); exit == 0 { return "zypper" }
    if _, _, exit, _ := c.Exec(ctx, "command -v apt-get >/dev/null", nil, false); exit == 0 { return "apt" }
    if _, _, exit, _ := c.Exec(ctx, "command -v rpm >/dev/null", nil, false); exit == 0 { return "rpm" }
    return "rpm"
}
