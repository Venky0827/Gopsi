## Overview
- Allow developers to build custom (third‑party) Gopsi modules and install them under a custom folder (e.g., `~/.gopsi/plugins/`).
- Loader scans this folder at startup, dynamically registers modules without rebuilding the Gopsi binary.
- Provide `gopsi module install/remove/list` commands to manage plugins locally (similar to ansible‑galaxy).

## Current Module Mechanism
- Registry and interface: `pkg/module/module.go:15-31` (Module interface, Register/Get).
- Parser: `pkg/play/parser.go:37-57` determines the module key from the task map.
- Runner: `pkg/runner/runner.go:126-136` fetches module by name and executes Validate/Check/Apply.
- Built‑ins are imported by CLI side‑effects: `cmd/gopsi/main.go:16-28`.

## Plugin Loader Design
- **Custom folder**: `~/.gopsi/plugins/` (override via `GOPSI_HOME`).
- **Format**: Go plugins (`.so`) built with `-buildmode=plugin` that export a `Register()` function.
- **Contract**: `Register()` must call `module.Register(mod{})` for each module implemented.
- **Loader**:
  - On startup, scan `~/.gopsi/plugins/*.so`.
  - For each plugin: `plugin.Open` and lookup `Register` symbol; invoke it.
  - Errors: log and continue; do not crash if a plugin fails.
- **Docs**:
  - Optional per‑plugin `help/*.md` files mapping `module_name.md` to manual text.
  - Loader reads docs into a `modhelp` registry so `gopsi module <name> help` includes plugin modules.

## CLI Management Commands
- `gopsi module install <git_url>`:
  - Clones repo into `~/.gopsi/plugins/src/<name>`.
  - Builds `plugin.so` with `go build -buildmode=plugin -o ~/.gopsi/plugins/<name>.so`.
  - Copies optional `help/*.md` to `~/.gopsi/plugins/help/<name>.md`.
- `gopsi module remove <name>`:
  - Deletes `~/.gopsi/plugins/<name>.so` and help file.
- `gopsi module list`:
  - Lists built‑in modules (`pkg/module.List()`) and plugin modules discovered at runtime.
- `gopsi module <name> help`:
  - First consult built‑in `modhelp`, then fallback to plugin help in `~/.gopsi/plugins/help/<name>.md`.

## Developer Workflow (Nutanix Example)
- Create repo: `github.com/yourorg/gopsi-nutanix`.
- Implement modules (e.g., `nutanix_vm`, `nutanix_image`) that integrate Prism APIs:
  - `Name() string` → `"nutanix_vm"`.
  - `Validate(args)` → ensure required args (`endpoint`, `username`, `password/token`, `action`, etc.).
  - `Check` → call Prism REST to determine current state and whether change is needed.
  - `Apply` → perform operation and return `Artifacts` (e.g., `vm_id`, `task_id`, `status`).
- Provide `func Register()` in the plugin that calls `module.Register(mod{})`.
- Build plugin: `go build -buildmode=plugin -o nutanix.so ./...`.
- Install: `gopsi module install https://github.com/yourorg/gopsi-nutanix` (CLI clones/builds and places `.so` under `~/.gopsi/plugins/`).
- Use in playbook:
  - `- name: power on VM\n    nutanix_vm: { name: app-01, action: power_on, endpoint: https://..., username: ..., password: ... }`.
  - `register: vmop` then `{{ vmop_artifacts.vm_id }}`.

## Idempotency & Security
- Idempotency: Ensure `Check` accurately reflects Prism state to avoid flapping.
- Secrets: Read credentials from play vars or environment; never log secrets. Use vault where possible.
- Verbosity: Respect `-v/-vv/-vvv` with helpful `Msg`/`Artifacts` but no sensitive data.

## Implementation Steps (Core)
- Add plugin loader to Gopsi:
  - Resolve plugins path: `GOPSI_HOME` or default `~/.gopsi` → `plugins/` and `help/` subfolders.
  - Load `.so` plugins and call their `Register()`.
  - Extend `modhelp` to read plugin help files at startup.
- Add CLI commands:
  - `module install/remove/list/help` handling.
  - Ensure informative errors and dry‑run modes if needed.
- Update docs with a “Developing Custom Modules” section explaining plugin authoring, packaging, and publishing.

## Testing Plan
- Build a sample plugin with a trivial module (e.g., `hello_world` returning a static message) and verify loader picks it up.
- Write unit tests for loader (mock filesystem with test plugins).
- Validate CLI commands for install/remove/list/help.
- Integration test: install Nutanix plugin and run sample play against a test Prism endpoint.

## Deliverables
- Plugin loader implementation with folder scan and help integration.
- CLI management commands for modules in the custom folder.
- Developer documentation (Modules.md section) for building and integrating third‑party modules.

## Notes
- Go plugins have tooling constraints (Go version compatibility, OS/arch). Document supported environments.
- Consider adding an alternative external‑process module interface later for language‑agnostic plugins.
- Keep JSON output clean; verbose prints suppressed in `--json`. 