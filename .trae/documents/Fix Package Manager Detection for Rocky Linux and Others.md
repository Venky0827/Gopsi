## Problem
- The `package` module incorrectly detects `apt` on Rocky Linux. Root cause: detection checks only for SSH command error, not shell exit status. `command -v apt-get` returns non-zero exit but no SSH error, leading to false detection.

## Plan
### 1) Correct Detection Logic
- Update `detectManager(ctx, c)` to check command exit codes (use `exit == 0`) instead of `err == nil`.
- Detection order prioritized for RHEL derivatives:
  1. `dnf`
  2. `yum`
  3. `apk`
  4. `zypper`
  5. `apt-get`
  6. `rpm` (fallback)
- Rationale: Rocky/RHEL should prefer `dnf`/`yum` over `apt`.

### 2) Use Detected Manager in Check/Apply
- In `Check`: select installed-state command by manager:
  - `apt`: `dpkg -s <name>`
  - `dnf|yum|zypper|rpm`: `rpm -q <name>`
  - `apk`: `apk info -e <name>`
- In `Apply`: use proper install/remove commands with `sudo -n`:
  - `apt`: `apt-get update -y && apt-get install -y` / `apt-get remove -y`
  - `dnf`: `dnf install -y` / `dnf remove -y`
  - `yum`: `yum install -y` / `yum remove -y`
  - `apk`: `apk add` / `apk del`
  - `zypper`: `zypper -n install -y` / `zypper -n remove -y`
  - `rpm`: `rpm -Uvh` / `rpm -e` (fallback)
- Keep artifacts including `manager`, `cmd`, `exit`.

### 3) Respect Become
- Ensure `become` flows into module args (already added in runner) and apply commands honor it.
- Verify `package` Apply uses `sudo -n` appropriately for all managers.

### 4) Unit Tests
- Add `pkg/modules/package/module_test.go` with a `fakeConn` that returns exit codes:
  - Rocky simulation: `dnf` present, `apt-get` absent → expect manager=`dnf`.
  - Ubuntu simulation: `apt-get` present → manager=`apt`.
  - Alpine: `apk` present → manager=`apk`.
- Test both `Check` detection and `Apply` command selection (by inspecting returned artifacts).

### 5) Verification
- Build and run a sample play on Rocky with `package: { name: tmux, action: install }` using `-vvv` to confirm manager is `dnf`/`yum` and command is correct.
- Confirm summary and colored outputs still behave per verbosity rules.

### 6) Safety
- No secrets printed; only args and artifacts.
- JSON mode remains unchanged (no verbose prints).

If approved, I will implement the detection fixes, add tests, and verify on your Rocky VM scenario. 