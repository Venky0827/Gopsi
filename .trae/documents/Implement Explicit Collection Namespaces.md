## Why You See "custom" Today
- We currently infer namespaces heuristically: anything registered before `loadPlugins()` is shown under `default`, anything registered after is shown under `custom`.
- There is no explicit collection metadata yet. The Nutanix plugin registers `nutanix_vm` during plugin load, so it falls into the "custom" bucket by definition.
- Code references: capture defaults (`cmd/gopsi/main.go:46–50`), compute custom by diff (`cmd/gopsi/main.go:331–336`), plugin loader scans `~/.gopsi/plugins` and calls exported `Register()` (`cmd/gopsi/main.go:432–458`).

## Proposed Namespace Model
- Add explicit collection namespace support in the registry so modules declare their collection.
- Backward compatible and minimal changes for built-ins and existing plugins.

## Technical Changes
1. Registry enhancements
- Add optional interface: `type Namespaced interface { Namespace() string }` in `pkg/module`.
- Track namespace per module in registry: default to `default` when absent.
- Provide `ListByNamespace() map[string][]string` to drive CLI output.

2. CLI updates
- Update `gopsi modules` to group by real namespaces: use `ListByNamespace()`.
- Keep fallback grouping: if no namespace declared and module was loaded via plugin, show under `custom`.

3. Plugin/Nutanix changes
- Implement `Namespace() string { return "nutanix" }` on the Nutanix module type.
- No change to `Register()` signature; existing `init()` registrations continue to work.

4. Optional metadata
- Support `CollectionName` exported symbol in plugins as an alternative to implementing the interface; loader reads it if present.
- Future: support `collection.yml` in a collection tarball with `name`, `version`, `dependencies`.

## Rollout Strategy
- Implement registry changes first; update CLI to use namespaces.
- Update Nutanix plugin to return `nutanix`.
- Keep current default/custom fallback for plugins without metadata.

## Verification
- Run `gopsi modules`; expect headings: `default` and `nutanix` (bold), with counts and grid.
- Ensure `nutanix_vm` appears under `nutanix`.
- Run unit tests to confirm no regressions.

## Next
- After approval, I will implement the registry interface, CLI grouping, and update the Nutanix plugin accordingly.