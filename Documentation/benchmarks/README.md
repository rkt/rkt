# Benchmarks

rkt has a utility called rkt-monitor that will run rkt with an example
workload, and track the memory and CPU usage. It does this by exec'ing rkt with
an ACI or pod manifest, watching the resource consumption for rkt and all
children processes, and after a timeout killing rkt and printing the results.

## Running the Benchmarks

To run the benchmarks, one must have both a built version of rkt-monitor and an
ACI or pod manifest. Additionally, rkt must be available on the `PATH`.

To build rkt-monitor, `cd` to `tests/rkt-monitor` and run the `build` script in
that directory.

To build one of the provided workloads, run any of the `build-*` scripts in
`tests/rkt-monitor`. All scripts require acbuild to be available on the current
`PATH`. The script will produce either an ACI, or a directory with multiple
ACIs and a pod manifest. In the case of the latter, the ACIs in the created
directory must be imported into rkt's cas before running rkt-monitor, via the
command `rkt fetch --insecure-options=image <newDirectory>/*`.

With rkt-monitor and an ACI or a pod manifest, now the benchmarks can be run
via `./rkt-monitor <workload>`.

There are two flags available to influence how rkt-monitor runs. `-v` will
print out the current resource usage of each process every second. `-d` can be
used to specify a duration to run the tests for (default of 10s). For example,
`-d 30s` will run the tests for 30 seconds.
