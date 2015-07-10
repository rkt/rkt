# rkt roadmap

**work in progress**

This document defines a high level roadmap for rkt development.
The dates below should not be considered authoritative, but rather indicative of the projected timeline of the project.
The [milestones defined in GitHub](https://github.com/coreos/rkt/milestones) represent the most up-to-date state of affairs.

rkt is an implementation of the [App Container spec](https://github.com/appc/spec), which is still under active development on an approximately similar timeframe.
Until the specification reaches version 1.0, rkt's major/minor versions will trail that of the spec.

### rkt 0.8 (July?)
- integrated alternative kvm stage1
- Smarter image discovery: cache control and etag handling 
- stable on-disk format
- stage1 split out from coreos/rkt
  - stable and documented stage0/stage1 ABI
- rkt can run applications as non-root

### rkt 0.9 (August?)
- rkt registers pods with systemd-machined where possible
  - machinectl {status,terminate,login,...}
- Better UX around trust
- rkt packaged for other distributions (e.g. Fedora)
- rkt can retrieve images via BitTorrent

### rkt 1.0 (September?)
- IPv6 support
