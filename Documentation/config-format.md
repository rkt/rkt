# Rocket Configuration File

Rocket reads configuration files with "json" extension residing in
/usr/lib/rkt/auth.d and /etc/rkt/auth.d/. From the filename you can
see it is just a JSON file. Please see an example below.

Currently there is only "auth" kind of configuration with only one
version ("v1"), which supports two types of authentication - "basic"
and "oauth". "basic" requires user name and password, "oauth" needs
OAuth Bearer Token. See examples below:

For basic auth:
```
{
	"rktKind": "auth",
	"rktVersion": "v1",
	"domains": ["domain1.com", "domain2.com"],
	"type": "basic",
	"credentials": {
		"user": "foo",
		"password": "bar"
	}
}
```

For oauth auth:
```
{
	"rktKind": "auth",
	"rktVersion": "v1",
	"domains": ["domain3.com", "domain4.com"],
	"type": "oauth",
	"credentials": {
		"token": "sometoken"
	}
}
```
