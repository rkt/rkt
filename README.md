# rkt - App Container runtime

[![godoc](https://godoc.org/github.com/coreos/rkt?status.svg)](http://godoc.org/github.com/coreos/rkt)
[![Build Status (Travis)](https://travis-ci.org/coreos/rkt.png?branch=master)](https://travis-ci.org/coreos/rkt)
[![Build Status (SemaphoreCI)](https://semaphoreci.com/api/v1/projects/28468e19-4fd0-483e-9c29-6c8368661333/395211/badge.svg)](https://semaphoreci.com/coreos/rkt)

![rkt Logo](logos/rkt-horizontal-color.png)

rkt (pronounced _"rock-it"_) is a CLI for running app containers on Linux. rkt is designed to be composable, secure, and fast.

Some of rkt's key features and goals include:

- First-class integration with init systems ([systemd][systemd], upstart) and cluster orchestration tools (fleet, [Kubernetes][kubernetes], [Nomad][nomad])
- Compatibility with other container software (e.g. rkt can run [Docker images][docker])
- Modular and extensible architecture ([network configuration plugins][networking], swappable execution engines based on systemd or [KVM][lkvm])

For more on the background and motivation behind rkt, read the original [launch announcement][blog-post].

[systemd]: Documentation/using-rkt-with-systemd.md
[kubernetes]: Documentation/using-rkt-with-kubernetes.md
[nomad]: Documentation/using-rkt-with-nomad.md
[docker]: Documentation/running-docker-images.md
[networking]: Documentation/networking.md
[lkvm]: Documentation/running-lkvm-stage1.md
[blog-post]: https://coreos.com/blog/rocket

## Project status

rkt is maturing rapidly while under active development, and its interfaces are not yet considered stable.
We do not recommend its use in production, but we encourage you to try out rkt and provide feedback via issues and pull requests.

Check out the [roadmap](ROADMAP.md) for more details on the future of rkt.

## rkt and App Container (appc)

rkt is an implementation of the [App Container (appc) spec](Documentation/app-container.md).
rkt's native image format ([ACI](Documentation/app-container.md#ACI)) and runtime/execution environment ([pods](Documentation/app-container.md#pods)) are defined in the specification.

## Trying out rkt

To get started quickly using rkt for the first time, start with the ["trying out rkt" document](Documentation/trying-out-rkt.md).
For an end-to-end example of building an application from scratch and running it with rkt, check out the [getting started guide](Documentation/getting-started-guide.md).

## Getting help with rkt

There are a number of different avenues for seeking help and communicating with the rkt community:
- For bugs and feature requests (including documentation!), file an [issue][new-issue]
- For general discussion about both using and developing rkt, join the [rkt-dev][rkt-dev] mailing list
- For real-time discussion, join us on IRC: #[rkt-dev][irc] on freenode.org
- For more details on rkt development plans, check out the GitHub [milestones][milestones]

Most discussion about rkt development happens on GitHub via issues and pull requests.
The rkt developers also host a semi-regular community sync meeting open to the public.
This sync usually features demos, updates on the roadmap, and time for anyone from the community to ask questions of the developers or share users stories with others.
For more details, including how to join and recordings of previous syncs, see the [sync doc on Google Docs][sync-doc].

[new-issue]: https://github.com/coreos/rkt/issues/new
[rkt-dev]: https://groups.google.com/forum/?hl=en#!forum/rkt-dev
[irc]: irc://irc.freenode.org:6667/#rkt-dev
[milestones]: https://github.com/coreos/rkt/milestones
[sync-doc]: https://docs.google.com/document/d/1NT_J5X2QErtKgd8Y3TFXNknWhJx_yOCMJnq3Iy2jPgE/edit#

## Contributing to rkt

rkt is an open source project under the Apache 2.0 [license](LICENSE), and contributions are gladly welcomed!
See the [Hacking Guide](Documentation/hacking.md) for more information on how to build and work on rkt.
See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting patches and the contribution workflow.

## Known issues

Due to a bug in the Linux kernel, using rkt's overlay support on top of an overlay filesystem requires Linux 4.3+.

