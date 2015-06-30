# rkt roadmap

**work in progress**

This document defines a high level roadmap for rkt development.
The dates below should not be considered authoritative, but rather indicative of the projected timeline of the project.
The [milestones defined in GitHub](https://github.com/coreos/rkt/milestones) represent the most up-to-date state of affairs.

rkt is an implementation of the [App Container spec](https://github.com/appc/spec), which is still under active development on an approximately similar timeframe.
Until the specification reaches version 1.0, rkt's major/minor versions will trail that of the spec.

### rkt 0.7 (June)
- new stage1 autoconf build system
- integrated alternative kvm stage1
- Smarter image discovery: cache control and etag handling 
- rkt registers pods with systemd-machined where possible
  - machinectl {status,terminate,login,...}
- Better UX around local image management with `rkt image`
 - extract, remove images
 - automatically fetch image dependencies
- Better UX around trust

### rkt 0.8-0.9 (July?)
- stable on-disk format
- stage1 split out from coreos/rkt
  - stable and documented stage0/stage1 ABI
- rkt packaged for other distributions (e.g. Fedora)
- rkt can run applications as non-root
- rkt can retrieve images via BitTorrent

### rkt 1.0 (July?)
- IPv6 support
