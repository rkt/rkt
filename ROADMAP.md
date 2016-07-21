# rkt roadmap

This document defines a high level roadmap for rkt development.
The dates below should not be considered authoritative, but rather indicative of the projected timeline of the project.
The [milestones defined in GitHub](https://github.com/coreos/rkt/milestones) represent the most up-to-date state of affairs.

rkt is an implementation of the [App Container spec](https://github.com/appc/spec), which is still under active development on an approximately similar timeframe.
The version of the spec that rkt implements can be seen in the output of `rkt version`.

rkt's version 1.0 release marks the command line user interface and on-disk data structures as stable and reliable for external development. The (optional) API for pod inspection is not yet completely stabilized, but is quite usable.

### rkt 1.12.0 (August)

- stable gRPC [API](https://github.com/coreos/rkt/tree/master/api/v1alpha)
- further improvements for SELinux environments, especially Fedora in enforcing mode
- enhanced DNS configuration [#2044](https://github.com/coreos/rkt/issues/2044)
- packaged for more distributions
  - CentOS [#1305](https://github.com/coreos/rkt/issues/1305)

### rkt 1.13.0 (August)

- `rkt fly` as top-level command [#1889](https://github.com/coreos/rkt/issues/1889)
- user configuration for stage1 [#2013](https://github.com/coreos/rkt/issues/2013)
