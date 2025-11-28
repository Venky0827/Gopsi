## Scope
- Implement a core set of Ansible-like modules as built-in Gopsi modules (registered via init and imported by the CLI), with consistent idempotent behavior, artifacts, and `become` support.
- Author a developer-facing document explaining usage in playbooks, arguments, examples, when-conditions, registers, artifacts, and best practices.

## Core Modules To Implement
1) File and Content
- `file`: ensure path present/absent, directories, permissions
- `copy`: transfer local file content to remote (checksum-based update)
- `template`: render and update remote (already present; add mode/owner/group)
- `lineinfile`: ensure a line present/absent in a file (regex-based)
- `blockinfile`: insert/update/remove named blocks in files

2) Commands
- `command`: run simple commands with guards `creates`/`removes` (already present)
- `shell`: run commands through shell with `become` support and env

3) Packages and Services
- `package`: auto-detect manager (apt/dnf/yum/apk/zypper/rpm) and install/remove (already implemented; refine detection/tests)
- `service`: start/stop/restart/enable/disable via systemd where available, fallback to sysvinit
- `yum_repository`/`apt_repository`: add/remove repos for package managers

4) Users and Groups
- `user`: create/remove users, manage shell, home, uid
- `group`: create/remove groups, manage gid

5) Network and Filesystem
- `get_url`: fetch remote URLs to files with checksum/ETag support
- `unarchive`: unpack archives (tar/zip) locally or remotely
- `mount`: mount/unmount filesystems; fstab entries
- `cron`: manage cron jobs

6) VCS and Language Tools
- `git`: clone/update repositories; branch/tag/ref
- `pip`: install Python packages (system or venv)

## Module Design & API
- Interface: `Validate`, `Check`, `Apply` per `pkg/module` contract
- Common arguments:
  - `become` (bool) honored where privileged ops required
  - `mode`, `owner`, `group` for file/template/copy
  - Idempotency: `Check` must predict change, `Apply` performs change and returns `Changed=true` only when change occurred
- Artifacts (visible in `-vvv` and available as `<register>_artifacts`): stdout/stderr/exit/cmd; before/after checksums; manager name; etc.

## Implementation Plan
- Create packages under `pkg/modules/<name>/module.go`
- Register each module in `init()` and import in `cmd/gopsi/main.go` for side effects
- Follow repo conventions (no secrets, sudo via `sudo -n`, consistent formatting)
- Ensure `runner` propagates `become` into module args (already done)

## Documentation Plan
- Create `docs/Modules.md` with sections:
  - Overview, common conventions, variables & `become`
  - Per-module reference: expected arguments, defaults, examples
  - Idempotency expectations and artifacts
  - Usage snippets within a playbook (YAML examples)
- Update `docs/Developer-Guide.md` table of contents to link `Modules.md`

## Testing & Validation
- Unit tests per module with `fakeConn` to simulate exits/outputs
- Golden tests for `lineinfile/blockinfile` modifications
- Integration: localhost or test VMs for apt/dnf/yum/apk/zypper
- Ensure `go test ./...` passes; verify `-v/-vv/-vvv` prints expected headers, timings, and artifacts

## Rollout
- Implement modules iteratively in this order for quick wins: `copy`, `shell`, `lineinfile`, `user`, `group`, `service` enhancements, `get_url`, `unarchive`, `git`, `pip`, `cron`, `mount`, repositories
- After implementation, add examples under `examples/` with playbook samples for each module

## Notes
- JSON mode: verbose lines suppressed to keep structured output clean
- Concurrency can interleave verbose lines; future enhancement: per-host prefixing or structured logs
- Backward compatibility: where meaningful (e.g., `package.state` mapped to `action`)