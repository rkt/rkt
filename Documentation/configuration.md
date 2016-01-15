# rkt configuration

`rkt` reads configuration from two or three directories - a **system directory**, a **local directory** and, if provided, a **user directory**.
The system directory defaults to `/usr/lib/rkt`, the local directory to `/etc/rkt`, and the user directory to an empty string.
These locations can be changed with command line flags described below.

The system directory should contain a configuration created by a vendor (e.g. distribution).
The contents of this directory should not be modified - it is meant to be read only.

The local directory keeps configuration local to the machine.
It can be modified by the admin.

The user directory may hold some user specific configuration.
It may be useful for specifying credentials used for fetching images without spilling them to some directory readable by everyone.

`rkt` looks for configuration files with the `.json` file name extension in subdirectories beneath the system and local directories.
`rkt` does not recurse down the directory tree to search for these files.
Users may therefore put additional appropriate files (e.g., documentation) alongside `rkt` configuration in these directories, provided such files are not named with the `.json` extension.

Every configuration file has two common fields: `rktKind` and `rktVersion`.
Both fields' values are strings, and the subsequent fields are specified by this pair.
The currently supported kinds and versions are described below.
These fields must be specified and cannot be empty.

`rktKind` describes the type of the configuration.
This is to avoid putting unrelated values into a single monolithic file.

`rktVersion` allows configuration versioning for each kind of configuration.
A new version should be introduced when doing some backward-incompatible changes: for example, when removing a field or incompatibly changing its semantics.
When a new field is added, a default value should be specified for it, documented, and used when the field is absent in any configuration file.
This way, an older version of `rkt` can work with newer-but-compatible versions of configuration files, and newer versions of `rkt` can still work with older versions of configuration files.

Configuration values in the system directory are superseded by the value of the same field if it exists in the local directory.
The same relationship exists between the local directory and the user directory if the user directory is provided.
The semantics of overriding configuration in this manner are specific to the `kind` and `version` of the configuration, and are described below.
File names are not examined in determining local overrides.
Only the fields inside configuration files need to match.

## Command line flags

To change the system configuration directory, use `--system-config` flag.
To change the local configuration directory, use `--local-config` flag.
To change the user configuration directory, use `--user-config` flag.

## Configuration kinds

### rktKind: `auth`

The `auth` configuration kind is used to set up necessary credentials when downloading images and signatures.
The configuration files should be placed inside the `auth.d` subdirectory (e.g., in the case of the default system/local directories, in `/usr/lib/rkt/auth.d` and/or `/etc/rkt/auth.d`).

#### rktVersion: `v1`

##### Description and examples

This version of the `auth` configuration specifies three additional fields: `domains`, `type` and `credentials`.

The `domains` field is an array of strings describing hosts for which the following credentials should be used.
Each entry must consist of a host/port combination in a URL as specified by RFC 3986.
This field must be specified and cannot be empty.

The `type` field describes the type of credentials to be sent.
This field must be specified and cannot be empty.

The `credentials` field is defined by the `type` field.
It should hold all the data that are needed for successful authentication with the given hosts.

This version of auth configuration supports two methods - basic HTTP authentication and OAuth Bearer Token.

Basic HTTP authentication requires two things - a user and a password.
To use this type, define `type` as `basic` and the `credentials` field as a map with two keys - `user` and `password`.
These fields must be specified and cannot be empty.
For example:

`/etc/rkt/auth.d/coreos-basic.json`:

```json
{
	"rktKind": "auth",
	"rktVersion": "v1",
	"domains": ["coreos.com", "tectonic.com"],
	"type": "basic",
	"credentials": {
		"user": "foo",
		"password": "bar"
	}
}
```

OAuth Bearer Token authentication requires only a token.
To use this type, define `type` as `oauth` and the `credentials` field as a map with only one key - `token`.
This field must be specified and cannot be empty.
For example:

`/etc/rkt/auth.d/coreos-oauth.json`:

```json
{
	"rktKind": "auth",
	"rktVersion": "v1",
	"domains": ["coreos.com", "tectonic.com"],
	"type": "oauth",
	"credentials": {
		"token": "sometoken"
	}
}
```

##### Override semantics

Overriding is done for each domain.
That means that the user can override authentication type and/or credentials used for each domain.
As an example, consider this system configuration:

`/usr/lib/rkt/auth.d/coreos.json`:

```json
{
	"rktKind": "auth",
	"rktVersion": "v1",
	"domains": ["coreos.com", "tectonic.com", "kubernetes.io"],
	"type": "oauth",
	"credentials": {
		"token": "common-token"
	}
}
```

If only this configuration file is provided, then when downloading data from either `coreos.com`, `tectonic.com` or `kubernetes.io`, `rkt` would send an HTTP header of: `Authorization: Bearer common-token`.

But with additional configuration provided in the local configuration directory, this can be overridden.
For example, given the above system configuration and the following local configuration:

`/etc/rkt/auth.d/specific-coreos.json`:

```json
{
	"rktKind": "auth",
	"rktVersion": "v1",
	"domains": ["coreos.com"],
	"type": "basic",
	"credentials": {
		"user": "foo",
		"password": "bar"
	}
}
```

`/etc/rkt/auth.d/specific-tectonic.json`:

```json
{
	"rktKind": "auth",
	"rktVersion": "v1",
	"domains": ["tectonic.com"],
	"type": "oauth",
	"credentials": {
		"token": "tectonic-token"
	}
}
```

The result is that when downloading data from `kubernetes.io`, `rkt` still sends `Authorization: Bearer common-token`, but when downloading from `coreos.com`, it sends `Authorization: Basic Zm9vOmJhcg==` (i.e. `foo:bar` encoded in base64).
For `tectonic.com`, it will send `Authorization: Bearer tectonic-token`.

Note that _within_ a particular configuration directory (either system or local), it is a syntax error for the same domain to be defined in multiple files.

##### Command line flags

There are no command line flags for specifying or overriding the auth configuration.

### rktKind: `dockerAuth`

The `dockerAuth` configuration kind is used to set up necessary credentials when downloading data from Docker registries.
The configuration files should be placed inside `auth.d` subdirectory (e.g. in `/usr/lib/rkt/auth.d` or `/etc/rkt/auth.d`).

#### rktVersion: `v1`

##### Description and examples

This version of `dockerAuth` configuration specifies two additional fields: `registries` and `credentials`.

The `registries` field is an array of strings describing Docker registries for which the associated credentials should be used.
This field must be specified and cannot be empty.
A short list of popular Docker registries is given below.

The `credentials` field holds the necessary data to authenticate against the Docker registry.
This field must be specified and cannot be empty.

Currently, Docker registries only support basic HTTP authentication, so `credentials` has two subfields - `user` and `password`.
These fields must be specified and cannot be empty.

Some popular Docker registries:

* index.docker.io (Assumed as the default when no specific registry is named on the rkt command line, as in `docker:///redis`.)
* quay.io
* gcr.io

Example `dockerAuth` configuration:

`/etc/rkt/auth.d/docker.json`:

```json
{
	"rktKind": "dockerAuth",
	"rktVersion": "v1",
	"registries": ["index.docker.io", "quay.io"],
	"credentials": {
		"user": "foo",
		"password": "bar"
	}
}
```

##### Override semantics

Overriding is done for each registry.
That means that the user can override credentials used for each registry.
For example, given this system configuration:

`/usr/lib/rkt/auth.d/docker.json`:

```json
{
	"rktKind": "dockerAuth",
	"rktVersion": "v1",
	"registries": ["index.docker.io", "gcr.io", "quay.io"],
	"credentials": {
		"user": "foo",
		"password": "bar"
	}
}
```

If only this configuration file is provided, then when downloading images from either `index.docker.io`, `gcr.io`, or `quay.io`, `rkt` would use user `foo` and password `bar`.

But with additional configuration provided in the local configuration directory, this can be overridden.
For example, given the above system configuration and the following local configuration:

`/etc/rkt/auth.d/specific-quay.json`:

```json
{
	"rktKind": "dockerAuth",
	"rktVersion": "v1",
	"registries": ["quay.io"],
	"credentials": {
		"user": "baz",
		"password": "quux"
	}
}
```

`/etc/rkt/auth.d/specific-gcr.json`:

```json
{
	"rktKind": "dockerAuth",
	"rktVersion": "v1",
	"domains": ["gcr.io"],
	"credentials": {
		"user": "goo",
		"password": "gle"
	}
}
```

The result is that when downloading images from `index.docker.io`, `rkt` still sends user `foo` and password `bar`, but when downloading from `quay.io`, it uses user `baz` and password `quux`; and for `gcr.io` it will use user `goo` and password `gle`.

Note that _within_ a particular configuration directory (either system or local), it is a syntax error for the same Docker registry to be defined in multiple files.

##### Command line flags

There are no command line flags for specifying or overriding the docker auth configuration.

### rktKind: `paths`

The `paths` configuration kind is used to customize the various paths that rkt uses.
The configuration files should be placed inside the `paths.d` subdirectory (e.g., in the case of the default system/local directories, in `/usr/lib/rkt/paths.d` and/or `/etc/rkt/paths.d`).

#### rktVersion: `v1`

##### Description and examples

This version of the `paths` configuration specifies one additional field: `data`.

The `data` field is a string that defines where image data and running pods are stored.
This field is optional.

Example `paths` configuration:

`/etc/rkt/paths.d/paths.json`:

```json
{
	"rktKind": "paths",
	"rktVersion": "v1",
	"data": "/home/me/rkt/data"
}
```

##### Override semantics

Overriding is done for each path.
For example, given this system configuration:

`/usr/lib/rkt/paths.d/data.json`:

```json
{
	"rktKind": "paths",
	"rktVersion": "v1",
	"data": "/opt/rkt-stuff/data"
}
```

If only this configuration file is provided, then rkt will store images and pods in the `/opt/rkt-stuff/data` directory.

But with additional configuration provided in the local configuration directory, this can be overridden.
For example, given the above system configuration and the following local configuration:

`/etc/rkt/paths.d/paths.json`:

```json
{
	"rktKind": "paths",
	"rktVersion": "v1",
	"data": "/home/me/rkt"
}
```

Now rkt will store the images and pods in the `/home/me/rkt` directory.
It will not know about any other data directory.

##### Command line flags

The `data` field can be overridden with the `--dir` flag.
