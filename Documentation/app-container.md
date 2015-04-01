## App Container basics

[App Container][appc-repo] is a [specification][appc-spec] of an image format, runtime, and discovery protocol for running applications in containers.

rkt implements the two runtime components of the specification: the [Application Container Executor (ACE)][appc-ace] and the [Metadata Service][appc-meta].

It also leverages schema and code from the upstream [appc/spec][appc-spec] repo to manipulate ACIs, work with image and pod manifests, and perform image discovery.

## Validating rkt

To validate that `rkt` successfully implements the ACE part of the spec, use the App Container [validation ACIs][appc-readme]:

```
$ sudo rkt --insecure-skip-verify run \
	--private-net --spawn-metadata-svc \
	--volume database,kind=host,source=/tmp \
	https://github.com/appc/spec/releases/download/v0.5.1/ace-validator-main.aci \
	https://github.com/appc/spec/releases/download/v0.5.1/ace-validator-sidekick.aci
```

[appc-repo]: https://github.com/appc/spec/
[appc-spec]: https://github.com/appc/spec/blob/master/SPEC.md
[appc-readme]: https://github.com/appc/spec/blob/master/README.md
[appc-ace]: https://github.com/appc/spec/blob/master/SPEC.md#app-container-executor
[appc-meta]: https://github.com/appc/spec/blob/master/SPEC.md#app-container-metadata-service
