# App Container 

## Overview

This repository contains schema definitions and tools for the App Container specification.
See [SPEC.md](SPEC.md) for details of the specification itself.
- `schema` contains JSON definitions of the different constituent formats of the spec (the _App Manifest_, the _Container Runtime Manifest_, and the `Fileset Manifest`). These JSON schemas also handle validation of the manifests through their Marshal/Unmarshal implementations.
  - `schema/types` contains various types used by the Manifest types to enforce validation
- `ace` contains a tool intended to be run within an _Application Container Executor_ to validate that the ACE has set up the container environment correctly. This tool can be built into an ACI image ready for running on an executor by using the `build_aci` script.
- `actool` contains a tool for building and validating images and manifests that meet the App Container specifications.

## Building ACIs 

`actool` can be used to build an Application Container Image from an application root filesystem (rootfs). It currently supports two modes: building an ACI from an existing [app manifest](SPEC.md#app-manifest), or building a [fileset image](SPEC.md#fileset-images) from a rootfs alone.

For example, to build a fileset containing certificate authorities, one could do the following:
```
$ actool build --fileset-name ca-certs /tmp/ca-certs/ ca-certs.aci
$ echo $?
0
```

Since an ACI is simply an (optionally compressed) tar file, we can inspect the created file with simple tools:

```
$ tar tvf ca-certs.aci
drwxrwxr-x 1000/1000         0 2014-01-02 03:04 rootfs/
drwxrwxr-x 1000/1000         0 2014-01-02 03:04 rootfs/certs/
-rw-rw-r-- 1000/1000      3140 2014-01-02 03:04 rootfs/certs/ca-bundle.crt
-rw-rw-r-- 1000/1000      3140 2014-01-02 03:04 rootfs/certs/ca-bundle.crt
-rw-rw-r-- 1000/1000      1581 2014-01-02 03:04 rootfs/certs/example.com.crt
-rw-r-xr-x root/root       174 2014-01-02 03:04 fileset
$ tar xf ca-certs.aci fileset -O | python -m json.tool
{
    "acKind": "FilesetManifest",
    "acVersion": "0.1.0",
    "arch": "amd64",
    "dependencies": null,
    "files": [
        "/certs/",
        "/ca-bundle.crt",
        "/example.com.crt",
    ],
    "name": "ca-certs",
    "os": "linux"
}
```

To build an ACI image containing an application, supply a valid app manifest and the rootfs:

```
; actool build --app-manifest my-app.json my_app/rootfs my-app.aci
```

Again, examining the ACI is simple, as is verifying that the app manifest was embedded appropriately:
```
$ tar tvf ca-certs.aci
drwxrwxr-x 1000/1000         0 2014-01-02 03:04 rootfs/
-rw-rw-r-- 1000/1000      1581 2014-01-02 03:04 rootfs/my_app
-rw-r-xr-x root/root       174 2014-01-02 03:04 app
```

```
; tar xf my-app.aci app -O | python -m json.tool
{
    "acKind": "AppManifest",
    "acVersion": "1.0.0",
    "arch": "amd64",
    "exec": [
        "/my_app",
    ],
    "group": "0",
    "name": "my_app",
    "os": "linux",
    "user": "0"
}
```

## Validating App Container implementations

`actool` can be used by implementations of the App Container Specification to check that files they produce conform to the expectations.

### Validating manifests

To validate one of the three manifest types in the specification, simply run `actool validate` against the file.

```
$ actool ./app.json
./app.json: valid AppManifest
$ echo $?
0
```

Multiple arguments are supported, and the output can be silenced with `-quiet`:

```
$ actool validate app1.json app2.json
app1.json: valid AppManifest
app2.json: valid AppManifest
$ actool -quiet validate app2.json
$ echo $?
0
```

`actool` will automatically determine which type of manifest it is checking (by using the `acKind` field common to all manifests), so there is no need to specify which type of manifest is being validated:
```
$ actool /tmp/my_fileset
/tmp/my_fileset: valid FilesetManifest
```

If a manifest fails validation, the first error encountered is returned, along with a non-zero exit status:
```
$ actool validate nover.json
nover.json: invalid AppManifest: acVersion must be set
$ echo $?
1
```

### Validating ACIs and layouts

Validating ACIs or layouts is very similar to validating manifests: simply run the `actool validate` subcommmand directly against an image or directory, and it will determine the type automatically:
```
$ actool validate bin/ace_validator_main.aci
ace_validator_main.aci: valid app container image
$ actool validate app_layout/
app_layout/: valid image layout
```

To override the type detection and force `actool validate` to consider a file as either a manifest or an image, use the `--type` flag:

```
actool validate -type appimage hello.aci
hello.aci: valid app container image
```

### Validating App Container Executors

TODO(jonboulle):...

