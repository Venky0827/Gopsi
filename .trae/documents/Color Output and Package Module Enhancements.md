## Goals
- Verbosity color rules: yellow when `changed=true`, green when `changed=false`, red on errors.
- Create/standardize a default `package` module that auto-detects Linux package manager and accepts `name` and `action`.

## Verbose Output Changes
- Update runner output in verbose mode:
  - Print a blank line before and after each task section (kept).
  - Use yellow for per-task summary lines when `changed=true`.
  - Use green for per-task summary lines when `changed=false`.
  - Keep red for validation/when/check/apply errors.
- Maintain clean `--json` mode (no color or verbose prints).
- Preserve final summary showing `success/total` and total runtime.

## Package Module (default builtin)
- Location: `pkg/modules/package` with module `Name() == "package"` and `init()` registration.
- Inputs:
  - `name` (string): package name
  - `action` (string): `install` or `remove` (default `install`)
- Auto-detect manager per host:
  - Managers supported: `apt-get`/`dpkg`, `yum`/`dnf`/`rpm`, `apk`, `zypper`.
  - Detection order via remote `Exec` checks (e.g., `command -v apt-get` etc.).
- Idempotent behavior:
  - `Check`: determines installed state using appropriate query (`dpkg -s`, `rpm -q`, `apk info -e`, `zypper se -i`).
  - `Apply`: runs the correct install/remove command using `sudo -n`.
- Artifacts:
  - Include `manager`, `cmd`, `installed_before`, `installed_after`, and `exit`.

## CLI Wiring
- Keep module imported by default in `cmd/gopsi/main.go` so users have it out-of-the-box.
- Backward compatibility:
  - Support existing `state` for now by mapping `present/absent` to `install/remove`, while documenting preferred `action`.

## Validation
- Build and run a sample play using `package:` tasks:
  - `- name: install curl\n  package: { name: curl, action: install }`
- Verify `-vv` prints yellow/green as specified and summary includes totals and duration.
- Run tests for parser and module compile; optional unit checks for detection helpers.

## Safety & Notes
- Do not print secrets; only show arguments and artifacts.
- Ensure error paths remain red and abort with clear messages.
- Keep JSON output unchanged for machine consumers.