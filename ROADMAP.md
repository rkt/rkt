# rkt roadmap

**work in progress**

This document defines a high level roadmap for rkt development.
The dates below should not be considered authoritative, but rather indicative of the projected timeline of the project.
The [milestones defined in GitHub](https://github.com/coreos/rkt/milestones) represent the most up-to-date state of affairs.

rkt is an implementation of the [App Container spec](https://github.com/appc/spec), which is still under active development on an approximately similar timeframe.
Until the specification reaches version 1.0, rkt's major/minor versions will trail that of the spec.

### rkt 0.9 (September)
- stable on-disk format [#1132](https://github.com/coreos/rkt/issues/1132)
- improved documentation around all top-level commands [#1164](https://github.com/coreos/rkt/issues/1164)
- rkt registers pods with systemd-machined where possible [#438](https://github.com/coreos/rkt/issues/438)
  - machinectl {status,terminate,login,...}
- better UX around trust [#367](https://github.com/coreos/rkt/issues/367)
- API service to expose read-only information about rkt pods [#1208](https://github.com/coreos/rkt/issues/1208)
- rkt runs on non-systemd distributions (Void Linux, RancherOS) [#1076](https://github.com/coreos/rkt/issues/1076)
- rkt packaged for other distributions 
  - Fedora [#1304](https://github.com/coreos/rkt/issues/1304) 
  - Gentoo [#1306](https://github.com/coreos/rkt/issues/1306) 
  - Debian [#1307](https://github.com/coreos/rkt/issues/1307) 

### rkt 1.0 (October?)
- rkt can retrieve images via BitTorrent [#405](https://github.com/coreos/rkt/issues/405)
- rkt packaged for more distributions
  - CentOS [#1305](https://github.com/coreos/rkt/issues/1305) 
  - OpenSUSE [#1304](https://github.com/coreos/rkt/issues/1308) 
- IPv6 support
- rkt can run applications as non-root [#820](https://github.com/coreos/rkt/issues/820)
