# rkt roadmap

This document defines a high level roadmap for rkt development.
The dates below should not be considered authoritative, but rather indicative of the projected timeline of the project.
The [milestones defined in GitHub](https://github.com/coreos/rkt/milestones) represent the most up-to-date state of affairs.

rkt is an implementation of the [App Container spec](https://github.com/appc/spec), which is still under active development on an approximately similar timeframe.
The version of the spec that rkt implements can be seen in the output of `rkt version`.

**Note that until communicated otherwise, the rkt CLI and API are considered unstable and subject to change.**
Backwards-compatibility is a priority for the v1.0.0 release.

### rkt v0.15 (January)
- Go 1.5 support [#1521](https://github.com/coreos/rkt/issues/1521)
- stage1 benchmarking [#1788](https://github.com/coreos/rkt/issues/1788)
- more advanced stage1 image configuration [#1425](https://github.com/coreos/rkt/issues/1425)
- per-user credentials configuration [#1568](https://github.com/coreos/rkt/issues/1568)

### rkt v0.16 (January)

- attach to the app's stdin/stdout [#1799](https://github.com/coreos/rkt/issues/1799)
- different shared namespace execution modes [#1433](https://github.com/coreos/rkt/issues/1433)		
  - `rkt fly`, suitable for running Kubelet		

### rkt 1.0
- stable CLI interface
- stable read-only API interface
- stable on-disk format (all upgrades should be backwards-compatible)
- fully integrated with `machinectl login` and `systemd-run` [#1463](https://github.com/coreos/rkt/issues/1463)
- packaged for more distributions
  - CentOS [#1305](https://github.com/coreos/rkt/issues/1305)
  - OpenSUSE [#1304](https://github.com/coreos/rkt/issues/1308)
  - Fedora [#1304](https://github.com/coreos/rkt/issues/1304)
  - Debian [#1307](https://github.com/coreos/rkt/issues/1307)
- rkt can run applications as non-root [#820](https://github.com/coreos/rkt/issues/820)
- IPv6 support [appc/cni#31](https://github.com/appc/cni/issues/31)
