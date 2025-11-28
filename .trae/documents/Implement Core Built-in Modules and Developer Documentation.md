## Goals
- Implement a practical set of Ansible-like core modules as first-class built-ins in Gopsi.
- Provide a clear developer document with playbook examples and usage guidance.

## Built-in Modules to Add
1) File & Content
- file: ensure path present/absent, directory handling, mode/owner/group
- copy: upload local content or file to remote with checksum-based idempotency
- template (enhance): support mode/owner/group; already present
- lineinfile: ensure a line is present/absent via regex
- blockinfile: insert/update/remove named blocks between markers

2) Commands
- command: run simple commands with guards (creates/removes); already present
- shell: run through a shell with env/support for pipelines and redirection

3) Packages & Services
- package: auto-detect manager (apt/dnf/yum/apk/zypper/rpm) and install/remove; already present (refine where needed)
- service: start/stop/restart/enable/disable via systemd, fallback to service; existing basic
- apt_repository/yum_repository: add/remove package repos (simple create/remove in sources.list.d or .repo)

4) Users & Groups
- user: present/absent; shell, home, uid, groups (simple create/delete)
- group: present/absent; gid

5) Network & Filesystem
- get_url: download a URL to a path with ETag/checksum
- unarchive: extract local/remote archives (tar/zip)
- mount: mount/unmount; fstab line management (basic present/absent)
- cron: manage cron entries for a user

6) VCS & Language Tools
- git: clone/update repo to a path, branch/ref
- pip: install/remove Python packages (system/venv)

## Module Contract & Conventions
- Every module implements Validate(args), Check(ctx, conn, args), Apply(ctx, conn, args)
- Idempotency: Check predicts change (Changed=true means Apply would change state)
- Common args
  - become: bool; honored by Apply when privileged ops needed (runner injects pl.Become)
  - mode/owner/group for file-like modules
  - name/action for package, service, user, group
- Artifacts: modules return structured `Artifacts` (stdout/stderr/exit/cmd, before/after checksum, manager/service state, etc.) for debugging and registers

## Implementation Details (high-level)
- file: `state: present|absent`, `path`, `mode`, `owner`, `group`; Check uses test -e; Apply uses mkdir/touch/rm; chown/chmod if needed
- copy: args `src|content`, `dest`, `mode`, `owner`, `group`; Check compares checksum; Apply uploads and sets perms/owner
- template: add perms/owner/group; unchanged rendering logic + checksum
- lineinfile: args `path`, `line`, `state`, `regexp` optional; Check/Apply with sed/awk or ed; backup optional
- blockinfile: args `path`, `block`, `marker`; ensure marker-wrapped block present/updated/removed
- shell: args `_` (string), `env`, `become`; run via `bash -lc`; guards same as command
- package: refine detection (exit-based), install/remove commands per manager
- service: add enable/disable; detect systemctl vs service; idempotent via is-active/is-enabled queries
- apt_repository/yum_repository: simple file drop/remove and yum makecache/apt update
- user/group: present/absent via `useradd/userdel` and `groupadd/groupdel`; Check via `/etc/passwd` and `/etc/group` lookup
- get_url: curl/wget with ETag; Check via checksum/mtime; Apply download
- unarchive: tar/zip extraction via `tar`/`unzip`; Check via marker/dir contents
- git: `git clone` or `git fetch/checkout`; Check via rev-parse; Apply update
- pip: `pip install --user` or venv; Check via `pip show`; Apply install/remove
- mount: query `mount` and `/etc/fstab`; Apply `mount/umount` and `fstab` line edits
- cron: manage entry lines in user crontab via `crontab -l` parsing

## CLI & Registration
- Place modules under `pkg/modules/<name>/module.go` and register via `init()`
- Ensure CLI imports built-in modules for side-effect registration so they’re available by default

## Documentation
- Create `docs/Modules.md` with:
  - Overview and conventions (become, artifacts, idempotency)
  - Per-module section: arguments, defaults, examples, artifacts
  - Playbook snippets demonstrating common patterns and `register`/`when` usage
  - Notes on platform differences (apt/dnf/yum/apk/zypper, systemd vs service)

## Testing
- Unit tests per module with `fakeConn` to simulate command responses
- Golden tests for lineinfile/blockinfile transformations
- Integration hints for running against local/VM targets

## Delivery Steps
1. Implement shell, copy, lineinfile, user, group (highest impact)
2. Wire CLI imports for registration
3. Draft docs/Modules.md with examples for the above
4. Implement remaining modules iteratively and expand docs
5. Run `go test ./...` and verify demos with `-vvv` verbosity

## Confirmation
- Once approved, I’ll implement these modules, wire them into the CLI, and add the documentation with examples.