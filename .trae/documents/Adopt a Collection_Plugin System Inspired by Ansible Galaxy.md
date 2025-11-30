## How Ansible Galaxy Collections Work
- Packaging:
  - Distribute as versioned tarballs with a `MANIFEST` and `galaxy.yml` (namespace, name, version, dependencies).
  - Content types: roles, modules, plugins, docs; referenced by `namespace.collection`.
- Installation:
  - `ansible-galaxy collection install <namespace.collection>:<version>` from Galaxy, or `-r requirements.yml`.
  - Installs into `~/.ansible/collections` or project’s `collections/` path; supports multiple versions side-by-side.
- Resolution:
  - Semantic versioning; dependencies resolved from `requirements.yml`.
  - Can install from Git URL, local tarballs, or Galaxy registry.
- Runtime:
  - Python drives plugin discovery via now-installed collection paths; modules executed by Ansible engine.
- Pros:
  - Easy distribution and versioning; dependency management; works offline with local tarballs.
  - Language-agnostic content (YAML roles, Python modules) packaged consistently.
- Cons:
  - Python runtime coupling; plugin discovery overhead.
  - Version conflicts and path precedence can be tricky.
  - Trust and provenance rely on Galaxy and tarball signatures; performance overhead for large collections.

## Gopsi Upgrade Goals
- Collections provide discoverable, versioned sets of modules, docs, and examples.
- Support installing from a registry or Git/URL tarballs; offline installs; keep built-ins separate.
- Minimize friction vs Go plugin constraints (Go version/OS). Consider hybrid: external executables or scripted modules.

## Proposed Architecture for Gopsi Collections
- Packaging:
  - Define `collection.yml` (name, namespace, version, dependencies, modules list).
  - Tarball layout: `modules/<name>/` (Go plugin `.so` or external executable), `help/<module>.md`, `examples/`, `LICENSE`.
- Installation Paths:
  - `~/.gopsi/collections/<namespace>/<name>/<version>/` for content.
  - Symlink or active pointer for “current” version.
- Module Types:
  - Go plugin `.so` (fast, in-process) with `Register()` symbol.
  - External module executables (JSON protocol over stdin/stdout) for language-agnostic support.
- Loader:
  - At startup, read `~/.gopsi/collections/**/current/`.
  - Load `.so` plugins via `plugin.Open` and `Register()`; register external executables via a shim provider.
  - Merge help docs into `modhelp` registry and `gopsi module <name> help`.
- Dependencies:
  - `collection.yml` lists dependencies with semver constraints; installer resolves and installs.
  - Implement basic resolver initially (no complex solver); fail clearly on conflicts.
- CLI Commands:
  - `gopsi galaxy install <namespace.name>[:version]` (registry) or `--from <tar|git>`.
  - `gopsi galaxy remove <namespace.name>`.
  - `gopsi galaxy list` (installed collections and versions).
  - `gopsi module list` (built-ins + loaded collection modules).
  - `gopsi module <name> help` (built-ins or collection docs).
- Security & Integrity:
  - Optional signature verification on tarballs (sha256 or signed index).
  - Trust policy: allow only user-installed content under `~/.gopsi`.
- Versioning Strategy:
  - Semantic versioning; allow multiple versions installed.
  - Default active version per collection; CLI to switch.

## Implementation Roadmap
- Phase 1 (MVP):
  - Define `collection.yml` schema and tarball structure.
  - Implement local installer for tarballs and Git; place under `~/.gopsi/collections`.
  - Extend loader to scan `collections/current` for `.so` and `help/*.md`.
  - Add `gopsi galaxy install/remove/list` and surface into `gopsi module list/help`.
- Phase 2 (Runtime + External Modules):
  - Add external-executable module shim with JSON protocol (for non-Go contributors).
  - Improve dependency resolution and version switching.
  - Caching and index support; optional registry endpoint.
- Phase 3 (Security & UX):
  - Tarball signature verification; trust policies.
  - Rich MAN pages generation, examples integration, and docs tooling.

## Developer Experience (Nutanix)
- Author a Nutanix collection:
  - `collection.yml` with `namespace: yourorg`, `name: nutanix`, `version: 0.1.0`.
  - Modules: `nutanix_vm`, `nutanix_image` (as `.so` or external executable).
  - Help files under `help/` and example plays under `examples/`.
- Build tarball and install:
  - `gopsi galaxy install yourorg.nutanix --from ./yourorg-nutanix-0.1.0.tar.gz`.
  - Use modules directly in playbooks; `gopsi module nutanix_vm help` shows docs.

## Why This Approach
- Combines Galaxy’s strengths (versioned, discoverable collections) with Gopsi’s needs:
  - Go plugins for performance; external executables for cross-language support.
  - Clear install location and separation from built-ins; safer upgrades.
  - Incremental path to a registry and richer ecosystem.

## Next Steps
- Implement the installer/loader and `collection.yml` parsing.
- Add external-executable shim and update module registry to support providers.
- Provide a sample Nutanix collection tarball and validate install/use end-to-end.
