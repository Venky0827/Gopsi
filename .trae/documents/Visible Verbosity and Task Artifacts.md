## Goals
- Make verbosity visible on stdout (not just stderr) with Ansible-like levels (-v, -vv, -vvv).
- Print task headers and step details so developers can see exactly what runs.
- Capture and expose per-task artifacts and store them under a predictable variable alongside `register`.

## Verbosity Design
- Levels:
  - -v: High-level execution flow
    - Print per-host connection details (user, addr, key path).
    - Print a task header before each task: `TASK [<name>] module=<module>`.
  - -vv: Add timing and resolved arguments
    - Print resolved task args (excluding `vars`).
    - Print Check/Apply durations and messages.
  - -vvv: Deep debug
    - Print `Result.Data` and `Result.Artifacts` (stdout/stderr/exit/cmd for `command`, checksums and paths for `template`, etc.).
    - Truncate very long values to keep output readable (e.g., 512 chars).

## Printing Strategy
- Route verbose output to stdout when `--json` is false so users see it immediately.
- Keep current `json` lines on stdout unchanged; verbose lines will be suppressed when `--json` is true to avoid mixing formats.
- Implement `Runner.verbose(level, msg)` that prints to stdout only when `r.verbosity >= level && !r.json`.
- Insert verbose prints in `pkg/runner/runner.go`:
  - Before SSH dial (connection summary).
  - Before task execution (task header with module).
  - After `Validate`, `When`, `Check` (with timing), and `Apply` (with timing).
  - At -vvv print artifacts via a helper that formats and truncates values.

## Artifacts & Register Variables
- `module.Result` already extended with `Artifacts map[string]any` (pkg/module/module.go:9-15).
- Ensure all core modules populate artifacts:
  - `command`: stdout, stderr, exit, cmd, sudo.
  - `file`: path, state, exists.
  - `template`: dest, before, after, mode.
  - `service`: name, state, active, cmd.
  - `package`: name, installed, cmd.
- Runner stores `register_artifacts` alongside `register`:
  - In `pkg/runner/runner.go`, when `t.Register != ""`, set `regs[t.Register+"_artifacts"] = res.Artifacts`.

## CLI Flags
- Maintain Ansible-like flags already present:
  - `-v`, `-vv`, `-vvv` mapped to verbosity levels 1..3 (cmd/gopsi/main.go:68-90, 140-145).
- Help text already reflects levels; add explicit note that verbose prints are shown in non-JSON mode.

## Output Examples
- -v:
  - `HOST localhost connect user=... addr=... key=...`
  - `TASK [run a command] module=command`
  - `localhost | run a command | changed=true`
- -vv:
  - Adds: `args={_:"touch files1", become:false}`
  - `check changed=true msg="" dur=12ms`
  - `apply changed=true msg="" dur=9ms`
- -vvv:
  - Adds: `artifacts: {cmd:"touch files1", exit:0, stdout:"", stderr:""}`

## Code Changes
- Update `pkg/runner/runner.go`:
  - Replace current `debugf` with `verbosef(level, ...)` printing to stdout when `!r.json`.
  - Add helper to format and truncate artifacts.
  - Insert prints at the key execution points as described.
- No change to module interfaces beyond the already-added `Artifacts` field.

## Validation
- Build and run with `-v`, `-vv`, `-vvv` and confirm visible differences on stdout.
- Ensure `--json` mode stays clean (no verbose lines).
- Run tests (`go test ./...`) to ensure modules compile and artifact attachments don’t break existing behavior.

## Notes
- Concurrency may interleave verbose lines; this is consistent with Ansible. Future enhancement: add per-host prefixes or structured logging.
- We avoid printing secrets by not dumping full `vars`; only show task args and artifacts produced by modules.
