# rkt configuration

`rkt` reads configuration from two directories - a **system directory** and a
**local directory**. The system directory defaults to `/usr/lib/rkt` and the
local directory `/etc/rkt`, but both can be overridden via command-line flags.

`rkt` looks for configuration files inside subdirectories of these two
directories. It ignores everything but regular files with `.json` extension.
This means `rkt` does _not_ search for the files by recursively going down
the directory tree. This also means that users are free to put some additional
files there (e.g. documentation).

Every configuration file has two common fields: `rktKind` and
`rktVersion`. Both are strings, and the rest of the fields are specified by
that pair. The currently supported kinds and versions are described below.
These fields must be specified and cannot be empty.

`rktKind` describes the type of the configuration. This is to avoid putting
unrelated values into single monolitic file.

`rktVersion` allows configuration versioning. A new version should be
introduced when doing some backward-incompatible changes: for example, when
removing a field or incompatibly changing its semantics. When a new field is
added, a default value should be specified for it, documented, and used when
the field is absent in file. This way, an older version of `rkt` can work with
newer-but-compatible versions of configuration files, and newer versions of
`rkt` can still work with older versions of configuration files.

The configuration in the system directory can be overridden by configuration
in the local directory. The semantics of configuration override are specific to
the kind and version of the configuration file and are described below.
Filenames do not play any role in overriding.

## Configuration kinds

### rktKind: `auth`

This kind of configuration is used to set up necessary credentials
when downloading images and signatures. The configuration files should
be placed inside the `auth.d` subdirectory (e.g., in the case of the default
system/local directories, in `/usr/lib/rkt/auth.d` or `/etc/rkt/auth.d`).

#### rktVersion: `v1`

##### Description and examples

This version of `auth` configuration specifies three additional
fields: `domains`, `type` and `credentials`.

The `domains` field is an array of strings describing hosts for which
following credentials should be used. By "host" we mean the host and
the port in a URL as specified by RFC 3986. This field has to be
specified and cannot be empty.

The `type` field describes the type of credentials to be sent. This
field has to be specified and cannot be empty.

`credentials` field is defined by the `type` field. It should hold all
the data that are needed for successful authentication with given
hosts.

This version of auth configuration supports two methods - basic HTTP
authentication and OAuth Bearer Token.

Basic HTTP authentication requires two things - a user and a
password. To use this type, define `type` field as `basic` and
`credentials` field as a map with two keys - `user` and
`password`. These fields have to be specified and cannot be
empty. Example:
```
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

OAuth Bearer Token authentication requires only a token. To use this
type, define `type` field as `oauth` and `credentials` field as a map
with only one key - `token`. This field has to be specified and cannot
be empty. Example:
```
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

##### Overriding semantics

Overriding is done for each domain. That means that the user can
override authentication type and/or credentials used for each
domain. Example of system configuration:

In `/usr/lib/rkt/auth.d/coreos.json`:
```
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

If only this configuration file were available to `rkt` then when
downloading data from either `coreos.com`, `tectonic.com` or
`kubernetes.io`, `rkt` would send an HTTP header: `Authorization:
Bearer common-token`.

But with additional configuration provided in the local configuration 
directory, this can be overridden. For example, given the above system
configuration and the following local configurations:

`/etc/rkt/auth.d/specific-coreos.json`:
```
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
```
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

the result is that when downloading data from `kubernetes.io`, `rkt`
still sends `Authorization: Bearer common-token`, but when downloading
from `coreos.com`, it sends `Authorization: Basic Zm9vOmJhcg==` (i.e 
`foo:bar` encoded in base64). For `tectonic.com`, it will send
`Authorization: Bearer tectonic-token`.

### rktKind: `dockerAuth`

This kind of configuration is used to set up necessary credentials
when downloading data from docker registries. The configuration files
should be placed inside `auth.d` subdirectory (that is, in
`/usr/lib/rkt/auth.d` or `/etc/rkt/auth.d`).

#### rktVersion: `v1`

##### Description and examples

This version of `dockerAuth` configuration specifies two additional
fields: `registries` and `credentials`.

The `registries` field is an array of strings describing docker
registries for which following credentials should be used. A short
list of popular docker registries is below. This field has to be
specified and cannot be empty.

`credentials` field holds the necessary data to authenticate against
docker registry. This field has to be specified and cannot be empty.

Currently docker registries only support basic HTTP authentication, so
`credentials` field has two subfields - `user` and `password`. These
fields have to be specified and cannot be empty.

Some popular docker registries:
* index.docker.io (this is is used when no docker registry is
  specified in URL, like in `docker://redis`)
* quay.io
* gcr.io

Example of dockerAuth config:
```
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

##### Overriding semantics

Overriding is done for each registry. That means that the user can
override credentials used for each registry. Example of system
configuration:

In `/usr/lib/rkt/auth.d/docker.json`:
```
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

If only this configuration file were available to `rkt` then when
downloading images from either `index.docker.io`, `gcr.io` or
`quay.io`, `rkt` would use user `foo` and password `bar`.

But with additional configuration like follows situation
changes. Example of local configuration:

In `/etc/rkt/auth.d/specific-quay.json`:
```
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
In `/etc/rkt/auth.d/specific-gcr.json`:
```
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

The result is that when downloading images from `index.docker.io`, 
`rkt` still sends user `foo` and password `bar`, but when downloading
from `quay.io`, it uses user `baz` and password `quux`; and for
`gcr.io` it will use user `goo` and password `gle`.
