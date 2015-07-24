# rkt functional tests

This directory contains a set of functional tests for rkt.
The tests use [gexpect](https://github.com/ThomasRooney/gexpect) to spawn various `rkt run` commands and look for expected output.

## Semaphore

The tests run on the [Semaphore](https://semaphoreci.com/) CI system through the [`rktbot`](https://semaphoreci.com/rktbot) user, which is part of the [`coreos`](https://semaphoreci.com/coreos/) org on Semaphore.
This user is authorized against the corresponding [`rktbot`](https://github.com/rktbot) GitHub account.
The credentials for `rktbot` are currently managed by CoreOS.

### Build settings

Use "Other" language and the following build commands:

```
./tests/install-deps.sh            # Setup
./tests/run-build.sh none          # Thread 1
./tests/run-build.sh src v222      # Thread 1
./tests/run-build.sh host          # Thread 1
./tests/run-build.sh coreos        # Thread 2
./tests/run-build.sh src master    # Thread 2
./tests/run-build.sh kvm           # Thread 2
git clean -ffdx                    # Post Thread
```

### Platform

Select `Ubuntu 14.04 LTS v1503 (beta with Docker support)`.

## Manually running the functional tests

Make sure that `--enable-functional-tests` is passed to configure
script, then, after building the project, functional tests can be run.

```
./configure --enable-functional-tests
make -j4
make check GO_TEST_FUNC_ARGS='-run NameOfTheTest'
```
