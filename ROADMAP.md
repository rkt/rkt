# rkt roadmap

**work in progress**

This document defines a high level roadmap for rkt development.
The dates below should not be considered authoritative, but rather indicative of the projected timeline of the project.
The [milestones defined in GitHub](https://github.com/coreos/rkt/milestones) represent the most up-to-date state of affairs.

rkt is an implementation of the [App Container spec](https://github.com/appc/spec), which is still under active development on an approximately similar timeframe.
Until the specification reaches version 1.0, rkt's major/minor versions will trail that of the spec.

### rkt 0.8 (July?)
- integrated alternative kvm stage1 [#1131](https://github.com/coreos/rkt/issues/1131)
- stable on-disk format [#1132](https://github.com/coreos/rkt/issues/1132)
- stable and documented stage0/stage1 ABI [#1133](https://github.com/coreos/rkt/issues/1133)
- Smarter image discovery: cache control and etag handling [#319](https://github.com/coreos/rkt/pull/319)
- stage1 split out from coreos/rkt (?) [#572](https://github.com/coreos/rkt/issues/572)

### rkt 0.9 (August?)
- rkt registers pods with systemd-machined where possible [#438](https://github.com/coreos/rkt/issues/438)
  - machinectl {status,terminate,login,...}
- Better UX around trust [#367](https://github.com/coreos/rkt/issues/367)
- rkt packaged for other distributions (e.g. Fedora) [#686](https://github.com/coreos/rkt/issues/686)
- rkt can retrieve images via BitTorrent [#405](https://github.com/coreos/rkt/issues/405)

### rkt 1.0 (September?)
- IPv6 support
- rkt can run applications as non-root [#820](https://github.com/coreos/rkt/issues/820)
