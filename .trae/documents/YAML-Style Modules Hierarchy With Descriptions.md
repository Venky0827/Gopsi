## Overview
- Replace the grid boxes with a YAML-style hierarchical listing.
- Show collection namespaces as bold headings.
- Under each namespace, list modules with a short one-line description.

## Data Sources for Descriptions
- Built-ins: parse `NAME` section from module docs in `pkg/modhelp` (e.g., "template - render template with vars...").
- Plugins: try to read `~/.gopsi/plugins/help/<module>.md` and parse `NAME` similarly.
- Fallback: if no doc is found, show a generic placeholder like "No description".

## Implementation Steps
1. Namespace Grouping
- Reuse current namespace grouping logic (built-ins → `default`, plugins → `custom` or declared `Namespace()`).
- Keep `Namespace()` interface support so collections like Nutanix display as `nutanix` instead of `custom`.

2. Description Extraction
- Add helper `shortDesc(name string) string`:
  - If `modhelp.Get(name)` returns doc, parse the `NAME` line to extract description after the module name.
  - Else read `plugins/help/<name>.md` under `gopsiHome()` and parse similarly.
  - Else return a default string.

3. YAML-Style Rendering
- For each namespace, print in bold: `namespace:`
- For each module in that namespace, print: `  - module: description`
- Order: `default` first, then other namespaces sorted alphabetically; modules sorted alphabetically within each namespace.

## Example Output
- Example for current setup:
  default:
    - command: run a shell command
    - copy: copy local or templated content to remote
    - template: render template with vars and copy to remote
    - ...
  nutanix:
    - nutanix_vm: manage Nutanix VM operations via Prism API

## Verification
- Run `gopsi modules` and confirm the YAML-style hierarchy with bold headers.
- Ensure modules without docs still appear with a sensible placeholder.
- No behavior change in `gopsi module list` flat output.

## Notes
- No changes required to the registry API beyond the optional `Namespace()` interface already introduced.
- This remains backward compatible with existing modules and plugins.