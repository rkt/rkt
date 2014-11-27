# app container format

This repository contains schema definitions and tools for the App Container specifications.
See [SPEC.md](SPEC.md) for details of the specifications themselves.
- `schema` contains JSON definitions of the different constituent formats of the spec (the _App Manifest_, the _Container Runtime Manifest_, and the `FileSet Manifest`). These JSON schemas also handle validation of the manifests through their Marshal/Unmarshal implementations.
  - `schema/types` contains various types used by the Manifest types to enforce validation
- `ace` contains a tool intended to be run within an _Application Container Executor_ to validate that the ACE has set up the container environment correctly. This tool can be built into an ACI image ready for running on an executor by using the `build_aci` script.
- `actool` contains a tool for building and validating images and manifests that meet the App Container specifications.

TODO(jonboulle): usage examples
- app-container/ace/build_aci
- bin/actool validate ...
- bin/actool build ...
