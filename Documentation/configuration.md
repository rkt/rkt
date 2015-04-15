# rkt configuration

`rkt` reads configuration from two directories - vendor one and custom
one. Vendor directory is by default `/usr/lib/rkt` and custom
directory - `/etc/rkt`. `rkt` looks for configuration files inside
subdirectories of those two. Inside them, it ignores everything but
regular files with `.json` extension. This means - `rkt` does _not_
search for the files by recursively going down the directory
tree. This also means that users are free to put some additional files
there as a documentation.

Every configuration file has two common fields: `rktKind` and
`rktVersion`. Both are strings. Rest of the fields are specified by
that pair. Currently supported kinds and versions are described
below. These fields have to be specified and cannot be empty.

Kind describes the type of the configuration. This is to avoid putting
unrelated values into single monolitic file.

Version allows configuration kind versioning. Please note that the new
version should be introduced when doing some backward-incompatible
changes. For example, when removing a field or incompatibly changing
its semantics. When a new field is added, some default value should be
specified for it, documented and used when the field is absent in
file. That way older version of `rkt` can work with
newer-but-compatible version of configuration files and newer versions
of `rkt` can still work with older versions of configuration files.

The configuration in vendor directory can be overridden by a
configuration in custom directory. Semantics of configuration override
are specific to the kind and the version of configuration file and are
described below. Filenames are not playing any role in overriding.

## kinds

### rktKind: `auth`

This kind of configuration is used to set up necessary credentials
when downloading images and signatures. The configuration files should
be placed inside `auth.d` subdirectory (that is - in
`/usr/lib/rkt/auth.d` or in `/etc/rkt/auth.d`).

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
domain. Example of vendor configuration:

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

But with additional configuration like follows situation
changes. Example of custom configuration:

In `/etc/rkt/auth.d/specific-coreos.json`:
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
In `/etc/rkt/auth.d/specific-tectonic.json`:
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

The result is that when downloading data from `kubernetes.io` `rkt`
still sends `Authorization: Bearer common-token`, but when downloading
from `coreos.com` - `Authorization: Basic Zm9vOmJhcg==` (`foo:bar`
encoded in base64). And for `tectonic.com` - `Authorization: Bearer
tectonic-token`.

### rktKind: `dockerAuth`

This kind of configuration is used to set up necessary credentials
when downloading data from docker indices. The configuration files
should be placed inside either `auth.d` or `docker.d` subdirectories
(that is - in `/usr/lib/rkt/{auth.d,docker.d}` or in
`/etc/rkt/{auth.d,docker.d}`).

#### rktVersion: `v1`

##### Description and examples

This version of `dockerAuth` configuration specifies two additional
fields: `indices` and `credentials`.

The `indices` field is an array of strings describing docker indices
for which following credentials should be used. A short list of
popular docker indices is below. This field has to be specified and
cannot be empty.

`credentials` field holds the necessary data to authenticate against
docker index. This field has to be specified and cannot be empty.

Currently docker indices only support basic HTTP authentication, so
`credentials` field has two subfields - `user` and `password`. These
fields have to be specified and cannot be empty.

Some popular docker indices:
* index.docker.io (this is is used when no docker index is specified
  in URL, like in `docker://redis`)
* quay.io
* gcr.io

Example of dockerAuth config:
```
{
	"rktKind": "dockerAuth",
	"rktVersion": "v1",
	"indices": ["index.docker.io", "quay.io"],
	"credentials": {
		"user": "foo",
		"password": "bar"
	}
}
```

##### Overriding semantics

Overriding is done for each index. That means that the user can
override credentials used for each index. Example of vendor
configuration:

In `/usr/lib/rkt/docker.d/docker.json`:
```
{
	"rktKind": "dockerAuth",
	"rktVersion": "v1",
	"indices": ["index.docker.io", "gcr.io", "quay.io"],
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
changes. Example of custom configuration:

In `/etc/rkt/auth.d/specific-quay.json`:
```
{
	"rktKind": "dockerAuth",
	"rktVersion": "v1",
	"indices": ["quay.io"],
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

The result is that when downloading images from `index.docker.io` `rkt`
still sends user `foo` and password `bar`, but when downloading
from `quay.io` - user `baz` and password `quux`. And for
`gcr.io` - user `goo` and password `gle`.
