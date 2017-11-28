# rkt roadmap

This document defines a high level roadmap for rkt development.
The dates below should not be considered authoritative, but rather indicative of the projected timeline of the project.
The [milestones defined in GitHub](https://github.com/rkt/rkt/milestones) represent the most up-to-date state of affairs.

rkt's version 1.0 release marks the command line user interface and on-disk data structures as stable and reliable for external development.

## Ongoing projects

### [Kubernetes CRI](https://github.com/rkt/rkt/projects/1)

Adapting rkt to offer first-class implementation of the Kubernetes Container Runtime Interface.

### [OCI native support](https://github.com/rkt/rkt/projects/4)

Supporting OCI specs natively in rkt.
Following OCI evolution and stabilization, it will become the preferred way over appc.
However, rkt will continue to support the ACI image format and distribution mechanism.
There is currently no plans to remove that support from rkt.

### Upcoming

Future tasks without a specific timeline are tracked at https://github.com/rkt/rkt/milestone/30.
