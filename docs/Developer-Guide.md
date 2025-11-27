# Gopsi Automation Tool Architecture and Developer Guide

## Overview
- Agent-less automation over SSH with YAML playbooks and inventories.
- Focus on simplicity, speed, and strong idempotency guarantees.
- Outputs human-friendly logs and optional JSON for integration.

## Directory Structure
- `cmd/at`: CLI entry (build output recommended as `gopsi`).
- `pkg/inventory`: Inventory loader and host/group resolution.
- `pkg/play`: Playbook model and parser.
- `pkg/conn`: SSH/SFTP connections and remote exec.
- `pkg/runner`: Orchestrates plays, tasks, concurrency, and output.
- `pkg/module`: Module interface and registry.
- `pkg/modules`: Builtin modules (`file`, `template`, `command`, `package`, `service`).
- `pkg/facts`: Remote facts gathering.
- `pkg/eval`: Safe evaluation for `when` conditionals.
- `pkg/vault`: Secrets encrypt/decrypt.
- `pkg/version`: Build and runtime version info.
- `examples`: Sample inventory and playbook.

## CLI Reference
- `gopsi run -i inventory.yml play.yml [--limit group] [--forks N] [--serial N] [--check] [--json]`
- `gopsi inventory --list -i inventory.yml`
- `gopsi vault --mode encrypt|decrypt --in file --out file --pass "..."`
- `gopsi version`

## Inventory Specification
- YAML root `all` with optional `children`, `hosts`, and `vars`.
- Host-level fields:
  - `host`: IP or DNS address
  - `user`: SSH username
  - `ssh_private_key_file`: path to private key
- Example:
```yaml
all:
  children:
    web:
      hosts:
        web1: { host: 10.0.0.11 }
        web2: { host: 10.0.0.12 }
  vars:
    user: deploy
    ssh_private_key_file: ~/.ssh/id_ed25519
```
- Variable precedence: play vars > host vars > group vars > inventory vars.
- `schema_version`: optional integer at root (default 1).

## Playbook Specification
- Either a list of plays or a map with `schema_version` and `plays` list.
- Play fields:
  - `hosts`: group or `all`
  - `become`: boolean
  - `serial`: rolling update batch size
  - `vars`: map
  - `tasks`: array of tasks
  - `handlers`: array of handler tasks
- Task fields:
  - `name`: human label
  - `module`: module key (by first map key other than standard fields)
  - `tags`: array of strings
  - `when`: conditional expression (`facts.os_family == "Linux"`, `not condition`)
  - `register`: variable name to store module result
  - `notify`: handler names to trigger

## Idempotent Modules
- Contract:
  - `Validate(args)` verifies the schema.
  - `Check(ctx, conn, args)` returns `Changed=true` if Apply would change state.
  - `Apply(ctx, conn, args)` performs changes and returns result.
- Builtins:
  - `file`: ensure path present/absent.
  - `template`: render locally and update remote when content changes.
  - `command`: run commands with guards (`creates`, `removes`).
  - `package`: install/remove via apt/yum.
  - `service`: systemd start/stop/restart.

## Facts and Conditionals
- Facts: OS family and distribution derived from remote.
- `when` evaluator supports simple equality and `not`.
- Extend evaluator to add logical ops, regex, and functions as needed.

## Execution Model
- Concurrency: `forks` controls parallelism; `serial` limits per play batch size.
- Check mode runs `Check` only and reports predicted changes.
- Handlers are triggered via `notify` and run after tasks.
- Output: human-friendly or `--json` per-task structured lines.

## Security
- Key-based SSH recommended; sudo uses non-interactive mode.
- Vault encrypts/decrypts secrets with AES-GCM.
- Avoid logging secrets; redact sensitive vars.
- Consider adding host key verification and known_hosts management.

## Performance
- Reuse SSH connections per host.
- Cache rendered templates and avoid unnecessary transfers using checksums.
- Tune `forks` and `serial` for fleet size and maintenance windows.

## Extending the Tool
- Add Modules:
  - Create a new package in `pkg/modules/<name>` implementing the Module interface.
  - Register in `init()`.
  - Ensure idempotent `Check` logic and deterministic outputs.
- Add Package/Service Adapters:
  - Detect managers and implement adapters (apk, dnf, zypper).
  - Select adapter based on facts.
- Enhance Evaluator:
  - Add logical operators, functions (e.g., `contains`, `toInt`), and numeric comparisons.
- Strategies and Output:
  - Add `strategy` styles or richer JSON schemas for downstream systems.

## Versioning and Migration
- `pkg/version` exposes `Version`, `Commit`, `Date`, `GoVersion` injected at build time.
- `schema_version` defaults to `1`; future changes should bump and validate.
- Semantic Versioning policy:
  - MAJOR: breaking changes (e.g., parser format changes).
  - MINOR: new features (modules, flags, evaluators).
  - PATCH: bug fixes and performance improvements.

## Testing
- Unit tests: parser, evaluator, module idempotency.
- Integration: localhost runs, Docker-based SSH targets.
- Golden tests for outputs where helpful.

## Developer Improvement Ideas
- Add roles and role dependencies for reusable automation blocks.
- Implement tag filtering and loops (`loop`, `with_items`).
- Add diff mode for file/template changes.
- Improve facts collection with hardware and network details.
- Add Windows support via WinRM and service/package adapters.
- Introduce connection pooling and retry/backoff strategies.
- Provide a plugin API for external modules with isolation.
