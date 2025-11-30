## Current Behavior
- `pkg/modules/file/module.go` only ensures presence/absence of a file at `path`.
- No support for writing content or setting permissions; `Validate` requires only `path`.
- `Check` uses `test -e` and `Apply` touches or removes the file.

## Desired Behavior
- Inputs: `file_name` (required), `path` (required), `state` (required: present|absent), `content` (required when present), `mode` (optional octal permissions).
- Behavior:
  - `present`: ensure directory exists, create the file `path/file_name`, write `content`, set `mode` if provided; idempotent via content checksum.
  - `absent`: remove `path/file_name` if exists.

## Technical Changes
1. Validation
- Require `file_name`, `path`, `state`.
- Enforce `state ∈ {present, absent}`.
- If `state=present`, require `content`.

2. Check Logic
- Compute `dest := filepath.Join(path, file_name)` (server-side path string).
- When `state=present`:
  - If file does not exist, `Changed=true`.
  - If exists and `content` provided, fetch remote file, compare SHA256 with `content`; set `Changed` accordingly.
- When `state=absent`:
  - If file exists, `Changed=true`; else `false`.
- Artifacts: `path`, `file_name`, `dest`, `exists`, `before`, `after`.

3. Apply Logic
- `absent`: `rm -rf dest`.
- `present`:
  - `mkdir -p $(dirname dest)`.
  - Upload `content` via `Conn.Put` and set permissions (`mode` if provided, default `0644`).
- Idempotent: only rewrites when content differs.

4. Rendering and Permissions
- Support templating in `path`, `file_name`, and `content` using existing `render()` helper.
- Add `parseOctal` helper (reuse pattern from `template/copy`) to parse `mode`.

5. Help Docs
- Update `pkg/modhelp/help.go` entry for `file`:
  - Args: `file_name` (required), `path` (required), `state` (present|absent), `content` (required when present), `mode` (optional octal).
  - Synopsis examples for presence with content and absence.

6. Tests
- Add tests in `pkg/modules/file`:
  - Validate errors for missing required fields.
  - `Check` detects change when file missing or content differs.
  - `Apply` writes content and sets mode; `absent` removes.

## Usage Examples
- Create a config file:
  - name: create cfg
    file: { path: /etc/app, file_name: app.cfg, state: present, content: "key=val", mode: "0644" }
- Remove a file:
  - name: remove cfg
    file: { path: /etc/app, file_name: app.cfg, state: absent }

## Rollout
- Implement module changes and docs.
- Run unit tests.
- Verify with a small playbook against a test host in check and apply modes.

## Backward Compatibility
- Existing `file: { path: /some/file }` usage will need to migrate to the new explicit args; we will document the change in the help entry and examples.

## Next Steps
- After approval, I will update the module code, docs, and tests accordingly, and validate end-to-end with a sample play.