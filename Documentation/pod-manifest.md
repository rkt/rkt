# Pod manifest walkthrough

The flag `--pod-manifest` can be passed to rkt's [run][run] and [prepare][prepare] subcommands.
It allows users to specify a *[runtime manifest][runtime-manifest]* to run as a pod.
A pod manifest completely specifies how the pod will be run, **overriding** any configuration present in the individual images of the apps in the pod.

Thus, by encoding how a rkt pod should be executed in a file, it can then be saved in version control and users don't need to deal with very long CLI arguments everytime they want to run a complicated pod.

## Generating a pod manifest

The most convenient way to generate a pod manifest is running a pod by using the rkt CLI (without `--pod-manifest`), exporting the resulting pod manifest, and then tweaking it until satisfied with it.

For example:

```bash
$ sudo rkt run coreos.com/etcd:v2.0.10 --- kinvolk.io/aci/busybox:1.24 -- -c 'while true; do date; sleep 1; done'
...
^]^]Container rkt-07f1cfdc-950b-4b6e-a2c0-8fb1ed37f98b terminated by signal KILL.
$ rkt cat-manifest 07f1cfdc > pod-manifest.json
```

The resulting pod manifest file is:

```json
{
	"acVersion": "1.29.0",
	"acKind": "PodManifest",
	"apps": [
		{
			"name": "etcd",
			"image": {
				"name": "coreos.com/etcd",
				"id": "sha512-c03b055d02e51e36f44a2be436eb77d5b0fbbbe37c00851188d8798912e8508a",
				"labels": [
					{
						"name": "os",
						"value": "linux"
					},
					{
						"name": "arch",
						"value": "amd64"
					},
					{
						"name": "version",
						"value": "v2.0.10"
					}
				]
			},
			"app": {
				"exec": [
					"/etcd"
				],
				"user": "0",
				"group": "0"
			}
		},
		{
			"name": "busybox",
			"image": {
				"name": "kinvolk.io/aci/busybox",
				"id": "sha512-140375b2a2bd836559a7c978f36762b75b80a7665e5d922db055d1792d6a4182",
				"labels": [
					{
						"name": "version",
						"value": "1.24"
					},
					{
						"name": "os",
						"value": "linux"
					},
					{
						"name": "arch",
						"value": "amd64"
					}
				]
			},
			"app": {
				"exec": [
					"sh",
					"-c",
					"while true; do date; sleep 1; done"
				],
				"user": "0",
				"group": "0",
				"ports": [
					{
						"name": "nc",
						"protocol": "tcp",
						"port": 1024,
						"count": 1,
						"socketActivated": false
					}
				]
			}
		}
	],
	"volumes": null,
	"isolators": null,
	"annotations": [
		{
			"name": "coreos.com/rkt/stage1/mutable",
			"value": "false"
		}
	],
	"ports": []
}
```

From there, you can edit the pod manifest following its [schema][pod-manifest-schema].

For example, we can add a memory isolator to etcd:

```json
...
				"exec": [
					"/etcd"
				],
				"isolators": [
					{
						"name": "resource/memory",
						"value": {"limit": "1G"}
					}
				],
...
```

Then, we can just run rkt with that pod manifest:

```
$ sudo rkt run --pod-manifest=pod-manifest.json
...
```

**Note** Images used by a pod manifest must be store in the local store, `--pod-manifest` won't do discovery or fetching.

Another option is running rkt with different CLI arguments until we have a configuration we like, and then just save the resulting pod manifest to use it later.

[run]: subcommands/run.md
[prepare]: subcommands/prepare.md
[runtime-manifest]: https://github.com/appc/spec/blob/v0.8.11/spec/pods.md#app-container-pods-pods
[pod-manifest-schema]: https://github.com/appc/spec/blob/v0.8.11/spec/pods.md#pod-manifest-schema
