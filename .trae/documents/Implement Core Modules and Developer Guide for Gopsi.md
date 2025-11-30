## Objectives

* Add a practical set of Ansible-like core modules as default built-ins, available out-of-the-box.

* Provide a concise developer guide with playbook usage examples, arguments, and best practices.

## Module Set

### File & Content

* file: Ensure path present/absent; supports `path`, `state`, `mode`, `owner`, `group` (idempotent via `test -e` and perms comparison).

* copy: Upload local `src` or inline `content` to remote `dest` with checksum-based idempotency; supports `mode`, `owner`, `group`.

* template (enhance): Add `mode`, `owner`, `group`; keep checksum-driven update.

* lineinfile: Ensure a `line` present/absent in `path`; optional `regexp` to replace matching line; idempotent via grep and sed.

* blockinfile: Manage a named block in `path` using markers; present/update/remove; idempotent via marker detection.

### Commands

* command: Already present; guards `creates`/`removes` for idempotency; `become`/`env` support.

* shell: Run via `bash -lc` for pipelines and redirection; supports `env`, `become`, guards.

### Packages & Services

* package: Implemented; auto-detect `apt/dnf/yum/apk/zypper/rpm` via exit codes; args `name`, `action (install|remove)`; idempotent via manager query.

* service (enhance): `name`, `state (started|stopped|restarted)`, `enabled (true|false)`; detect systemd vs service; idempotent via `is-active`/`is-enabled`.

* apt\_repository/yum\_repository: Add/remove simple repo file entries; run apt update or yum/dnf makecache on change.

### Users & Groups

* user: `name`, `state (present|absent)`, optional `uid`, `shell`, `home`, `groups`; idempotent via `/etc/passwd`.

* group: `name`, `state (present|absent)`, optional `gid`; idempotent via `/etc/group`.

### Network & Filesystem

* get\_url: `url`, `dest`, optional `checksum`; idempotent via ETag/checksum; uses curl/wget.

* unarchive: `src`, `dest`, `remote_src` toggle; supports tar/zip; idempotent via extracted marker/check.

* mount: `path`, `src`, `fstype`, `state (mounted|unmounted|present|absent)`; idempotent via `mount` and `/etc/fstab` checks.

* cron: Manage cron entries `name`, `user`, `minute/hour/...`, `job`, `state`; idempotent via parsing crontab.

### VCS & Language Tools

* git: `repo`, `dest`, optional `version`, `force`; idempotent via `rev-parse` and working tree.

* pip: `name`, `state (present|absent)`, optional `virtualenv`; idempotent via `pip show`.

## Module Interface & Conventions

* All modules implement `Validate(args)`, `Check(ctx, conn, args)`, `Apply(ctx, conn, args)`.

* Common features:

  * `become`: Runner injects `pl.Become`; modules honor it for privileged ops (sudo -n).

  * Artifacts: Return structured details (e.g., `stdout`, `stderr`, `exit`, `cmd`, `before/after checksums`, `manager`).

  * Idempotency: Check predicts state change; Apply only returns `Changed=true` when actual change occurred.

## Developer Guide (docs/Modules.md)

* Overview: module contract, idempotency, `become`, variables, `register` and `_artifacts` usage.

* Per-module sections: arguments (required/optional), defaults, common examples, artifacts returned, and `when` examples.

* Best practices: permissions, environment safety, cross-distro notes (package/service), and JSON mode behavior.

* Link from existing Developer-Guide to Modules.md.

## Implementation Steps

1. Implement shell, copy, lineinfile, user, group modules under `pkg/modules/<name>/module.go` with `init()` registration.
2. Enhance template/service; add apt/yum repository modules.
3. Implement get\_url, unarchive, cron, mount, git, pip.
4. Import new modules in CLI (`cmd/gopsi/main.go`) for side-effect registration.
5. Write unit tests using `fakeConn` to simulate exits/outputs; add golden tests for line/block modifications.
6. Create `docs/Modules.md` with examples and usage; update Developer-Guide index.

## Validation

* `go build` and `go test ./...` pass.

* Run example plays for each module with `-vvv` to verify headers, timings, artifacts, and idempotent behavior.

* Confirm JSON mode remains clean (verbose suppressed).

## Timeline & Priorities

* Phase 1: shell, copy, lineinfile, user, group, service enhancements.

* Phase 2: get\_url, unarchive, cron, mount, git, pip, repositories.

* Phase 3: doc polish and more examples.

## Notes

* Avoid printing secrets; redact sensitive vars.

* Support cross-distro detection where applicable; prefer exit-based checks over errors.

* Future: roles, loops, tag filtering, Windows support.

