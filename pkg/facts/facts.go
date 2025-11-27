package facts

import (
    "context"
    "strings"

    "gopsi/pkg/module"
)

type Facts map[string]any

func Gather(ctx context.Context, c module.Conn) (Facts, error) {
    out, _, _, err := c.Exec(ctx, "uname -s", nil, false)
    if err != nil { return nil, err }
    osfam := "Linux"
    if strings.Contains(out, "Darwin") { osfam = "Darwin" }
    facts := Facts{"os_family": osfam}
    etc, _, _, _ := c.Exec(ctx, "cat /etc/os-release || true", nil, false)
    if strings.Contains(etc, "Ubuntu") { facts["distro"] = "Ubuntu" }
    if strings.Contains(etc, "Debian") { facts["distro"] = "Debian" }
    if strings.Contains(etc, "CentOS") || strings.Contains(etc, "Red Hat") { facts["distro"] = "RHEL" }
    return facts, nil
}
