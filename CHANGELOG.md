### v0.8.0

rkt 0.8.0 includes support for running containers under an LKVM hypervisor
and experimental user namespace support.

Full changelog:

- Documentation improvements
- Better integration with systemd:
 - journalctl -M
 - machinectl {reboot,poweroff}
- Update stage1's systemd to v222
- Add more functional tests
- Build system improvements
- Fix bugs with garbage-collection
- LKVM stage1 support with network and volumes
- Smarter image discovery: ETag and Cache-Control support
- Add CNI DHCP plugin
- Support systemd socket activation
- Backup CAS database when migrating
- Improve error messages
- Add the ability to override ACI exec
- Optimize rkt startup times when a stage1 is present in the store
- Trust keys fetched via TLS by default
- Add the ability to garbage-collect a specific pod
- Add experimental user namespace support
- Bugfixes

### v0.7.0

rkt 0.7.0 includes new subcommands for `rkt image` to manipulate images from
the local store.

It also has a new build system based on autotools and integration with SELinux.

Full changelog:

- New subcommands for `rkt image`: extract, render and export
- Metadata service:
  - Auth now based on tokens
  - Registration done by default, unless --mds-register=false is passed
- Build:
  - Remove support for Go 1.3
  - Replace build system with autoconf and make
- Network: fixes for plugins related to mnt namespace
- Signature: clearer error messages
- Security:
  - Support for SELinux
  - Check signature before downloading
- Commands: fix error messages and parameter parsing
- Output: reduce output verbosity
- Systemd integration: fix stop bug
- Tests: Improve tests output

### v0.6.1

The highlight of this release is the support of per-app memory and CPU
isolators. This means that, in addition to restricting a pod’s CPU and memory
usage, individual apps inside a pod can also be restricted now.

rkt 0.6.1 also includes a new CLI/subcommand framework, more functional testing
and journalctl integration by default.

Full changelog:

* Updated to v0.6.1 of the appc spec
* support per-app memory and CPU isolators
* allow network selection to the --private-net flag which can be useful for
  grouping certain pods together while separating others
* move to the Cobra CLI/subcommand framework
* per-app logging via journalctl now supported by default
* stage1 runs an unpatched systemd v220
* to help packagers, rkt can generate stage1 from the binaries on the host at
  runtime
* more functional tests
* bugfixes

### v0.5.6

rkt 0.5.6 includes better integration with systemd on the host, some minor bug
fixes and a new ipvlan network plugin.

- Updated to v0.5.2 of the appc spec
- support running from systemd unit files for top-level isolation
- support per-app logging via journalctl. This is only supported if stage1 has
  systemd v219 or v220
- add ipvlan network plugin
- new rkt subcommand: cat-manifest
- extract ACI in a chroot to avoid malformed links modifying the host
  filesystem
- improve rkt error message if the user doesn’t provide required volumes
- fix rkt status when using overlayfs
- support for some arm architectures
- documentation improvements


### v0.5.5

rkt 0.5.5 includes a move to [cni](https://github.com/appc/cni) network
plugins, a number of minor bug fixes and two new experimental commands for
handling images: `rkt images` and `rkt rmimage`. 

Full changelog:
- switched to using [cni](https://github.com/appc/cni) based network plugins
- fetch images dependencies recursively when ACIs have dependent images
- fix the progress bar used when downloading images with no content-length
- building the initial stage1 can now be done on various versions of systemd
- support retrying signature downloads in the case of a 202
- remove race in doing a rkt enter
- various documentation fixes to getting started and other guides
- improvements to the functional testing using a new gexpect, testing for
  non-root apps, run context, port test, and more


### v0.5.4

rkt 0.5.4 introduces a number of new features - repository authentication,
per-app arguments + local image signature verification, port forwarding and
more. Further, although we aren't yet guaranteeing API/ABI stability between
releases, we have added important work towards this goal including functional
testing and database migration code.

This release also sees the removal of the `--spawn-metadata-svc` flag to 
`rkt run`. The flag was originally provided as a convenience, making it easy
for users to get started with the metadata service.  In rkt v0.5.4 we removed
it in favor of explicitly starting it via `rkt metadata-service` command. 

Full changelog:
- added configuration support for repository authentication (HTTP Basic Auth,
  OAuth, and Docker repositories). Full details in
  `Documentation/configuration.md`
- `rkt run` now supports per-app arguments and per-image `--signature`
  specifications
- `rkt run` and `rkt fetch` will now verify signatures for local image files
- `rkt run` with `--private-net` now supports port forwarding (using
  `--port=NAME:1234`)
- `rkt run` now supports a `--local` flag to use only local images (i.e. no
  discovery or remote image retrieval will be performed)
- added initial support for running directly from a pod manifest
- the store DB now supports migrations for future versions
- systemd-nspawn machine names are now set to pod UUID
- removed the `--spawn-metadata-svc` option from `rkt run`; this mode was
  inherently racy and really only for convenience. A separate 
  `rkt metadata-service` invocation should be used instead.
- various internal codebase refactoring: "cas" renamed to "store", tasks to
  encapsulate image fetch operations, etc
- bumped docker2aci to support authentication for Docker registries and fix a
  bug when retrieving images from Google Container Registry
- fixed a bug where `--interactive` did not work with arguments
- garbage collection for networking is now embedded in the stage1 image
- when rendering images into the treestore, a global syncfs() is used instead
  of a per-file sync(). This should significantly improve performance when
  first extracting large images
- added extensive functional testing on semaphoreci.com/coreos/rkt
- added a test-auth-server to facilitate testing of fetching images


### v0.5.3
This release contains minor updates over v0.5.2, notably finalising the move to
pods in the latest appc spec and becoming completely name consistent on `rkt`.
- {Container,container} changed globally to {Pod,pod}
- {Rocket,rocket} changed globally to `rkt`
- `rkt install` properly sets permissions for all directories
- `rkt fetch` leverages the cas.Store TmpDir/TmpFile functions (now exported)
  to generate temporary files for downloads
- Pod lifecycle states are now exported for use by other packages
- Metadata service properly synchronizes access to pod state


### v0.5.2

This release is a minor update over v0.5.1, incorporating several bug fixes and
a couple of small new features:
- `rkt enter` works when overlayfs is not available
- `rkt run` now supports the `--no-overlay` option referenced (but not
  implemented!) in the previous release
- the appc-specified environment variables (PATH, HOME, etc) are once again set
  correctly during `rkt run`
- metadata-service no longer manipulates IP tables rules as it connects over a
  unix socket by default
- pkg/lock has been improved to also support regular (non-directory) files
- images in the cas are now locked at runtime (as described in #460)


### v0.5.1

This release updates Rocket to follow the latest version of the appc spec,
v0.5.1. This involves the major change of moving to _pods_ and _Pod Manifests_
(which enhance and supplant the previous _Container Runtime Manifest_). The
Rocket codebase has been updated across the board to reflect the schema/spec
change, as well as changing various terminology in other human-readable places:
for example, the previous ambiguous (unqualified) "container" is now replaced
everywhere with "pod".

This release also introduces a number of key features and minor changes:
- overlayfs support, enabled for `rkt run` by default (disable with
  `--no-overlayfs`)
- to facilitate overlayfs, the CAS now features a tree store which stores
  expanded versions of images
- the default stage1 (based on systemd) can now be built from source, instead
  of only derived from an existing binary distribution as previously. This is
  configurable using the new `RKT_STAGE1_USR_FROM` environment variable when
  invoking the build script - see fdcd64947
- the metadata service now uses a Unix socket for registration; this limits who
  can register/unregister pods by leveraging filesystem permissions on the
  socket
- `rkt list` now abbreviates UUIDs by default (configurable with `--full`)
- the ImageManifest's `readOnly` field (for volume mounts) is now overridden by
  the rkt command line
- a simple debug script (in scripts/debug) to facilitate easier debugging of
  applications running under Rocket by injecting Busybox into the pod
- documentation for the metadata service, as well as example systemd unit files


### v0.4.2

- First support for interactive containers, with the `rkt run --interactive`
  flag. This is currently only supported if a container has one app. #562 #601 
- Add container IP address information to `rkt list`
- Provide `/sys` and `/dev/shm` to apps (per spec)
- Introduce "latest" pattern handling for local image index
- Implement FIFO support in tar package
- Restore atime and mtime during tar extraction
- Bump docker2aci dependency


### v0.4.1

This is primarily a bug fix release with the addition of the `rkt install`
subcommand to help people setup a unprivileged `rkt fetch` based on unix users.

- Fix marshalling error when running containers with resource isolators
- Fixup help text on run/prepare about volumes
- Fixup permissions in `rkt trust` created files
- Introduce the `rkt install` subcommand


### v0.4.0

This release is mostly a milestone release and syncs up with the latest release
of the [appc spec](https://github.com/appc/spec/releases/tag/v0.4.0) yesterday.

Note that due to the introduction of a database for indexing the local CAS,
users upgrading from previous versions of Rocket on a system may need to clear
their local cache by removing the `cas` directory. For example, using the
standard Rocket setup, this would be accomplished with 
`rm -fr /var/lib/rkt/cas`.

Major changes since v0.3.2:
- Updated to v0.4.0 of the appc spec
- Introduced a database for indexing local images in the CAS (based on
  github.com/cznic/ql)
- Refactored container lifecycle to support a new "prepared" state, to
- pre-allocate a container UUID without immediately running the application
- Added support for passing arguments to apps through the `rkt run` CLI
- Implemented ACI rendering for dependencies
- Renamed `rkt metadatasvc` -> `rkt metadata-service`
- Added documentation around networking, container lifecycle, and rkt commands


### v0.3.2

This release introduces much improved documentation and a few new features.

The highlight of this release is that Rocket can now natively run Docker
images. To do this, it leverages the appc/docker2aci library which performs a
straightforward conversion betwen images in the Docker format and the appc
format.

A simple example:

```
$ rkt --insecure-skip-verify run docker://redis docker://tenstartups/redis-commander
rkt: fetching image from docker://redis
rkt: warning: signature verification has been disabled
Downloading layer: 511136ea3c5a64f264b78b5433614aec563103b4d4702f3ba7d4d2698e22c158
```

Note that since Docker images do not support image signature verifications, the
`-insecure-skip-verify` must be used.

Another important change in this release is that the default location for the
stage1 image used by `rkt run` can now be set at build time, by setting the
`RKT_STAGE1_IMAGE` environment variable when invoking the build script. (If
this is not set, `rkt run` will continue with its previous behaviour of looking
for a stage1.aci in the same directory as the binary itself. This makes it
easier for distributions to package Rocket and include the stage1 wherever
they choose (for example, `/usr/lib/rkt/stage1.aci`). For more information, see
https://github.com/coreos/rocket/pull/520


### v0.3.1

The primary motivation for this release is to resynchronise versions with the
appc spec. To minimise confusion in the short term we intend to keep the
major/minor version of Rocket aligned with the version of spec it implements;
hence, since yesterday v0.3.0 of the appc spec was released, today Rocket
becomes v0.3.1. After the spec (and Rocket) reach v1.0.0, we may relax this
restriction.

This release also resolves an upstream bug in the appc discovery code which was
causing rkt trust to fail in certain cases.


### v0.3.0

This is largely a momentum release but it does introduce a few new user-facing
features and some important changes under the hood which will be of interest to
developers and distributors.

First, the CLI has a couple of new commands:
- `rkt trust` can be used to easily add keys to the public keystore for ACI
  signatures (introduced in the previous release). This supports retrieving
  public keys directly from a URL or using discovery to locate public keys - a
  simple example of the latter is `rkt trust --prefix coreos.com/etcd`. See the
  commit for other examples.
- `rkt list` is an extremely simple tool to list the containers on the system

As mentioned, v0.3.0 includes two significant changes to the Rocket build process:
- Instead of embedding the (default) stage1 using go-bindata, Rocket now
  consumes a stage1 in the form of an actual ACI, containing a rootfs and
  stage1 init/exec binaries. By default, Rocket will look for a `stage1.aci` in
  the same directory as the location of the binary itself, but the stage1 can
  be explicitly specified with the new `-stage1-image` flag (which deprecates
  `-stage1-init` and `-stage1-rootfs`). This makes it much more straightforward
  to use alternative stage1 images with rkt and facilitates packing it for
  different distributions like Fedora.
- Rocket now vendors a copy of the appc/spec instead of depending on HEAD. This
  means that Rocket can be built in a self-contained and reproducible way and
  that master will no longer break in response to changes to the spec. It also
  makes explicit the specific version of the spec against which a particular
  release of Rocket is compiled.

As a consequence of these two changes, it is now possible to use the standard
Go workflow to build the Rocket CLI (e.g. `go get github.com/coreos/rocket/rkt`
will build rkt). Note however that this does not implicitly build a stage1, so
that will still need to be done using the included ./build script, or some
other way for those desiring to use a different stage1.


### v0.2.0

This introduces countless features and improvements over v0.1.1. Highlights
include several new commands (`rkt status`, `rkt enter`, `rkt gc`) and
signature validation.


### v0.1.1

The most significant change in this release is that the spec has been split
into its own repository (https://github.com/appc/spec), and significantly
updated since the last release - so many of the changes were to update to match
the latest spec.

Numerous improvements and fixes over v0.1.0:
- Rocket builds on non-Linux (in a limited capacity)
- Fix bug handling uncompressed images
- More efficient image handling in CAS
- mkrootfs now caches and GPG checks images
- stage1 is now properly decoupled from host runtime
- stage1 supports socket activation
- stage1 no longer warns about timezones
- cas now logs download progress to stdout
- rkt run now acquires an exclusive lock on the container directory and records
  the PID of the process


### v0.1.0

- tons of documentation improvements added
- actool introduced along with documentation
- image discovery introduced to rkt run and rkt fetch


### v0.0.0

Initial release.
