# ACI hosting

rkt uses [App Container Images (ACIs)][ACI] as the native packaging format for [application containers][[application-container].
To distribute those images, the appc spec defines an [Image Discovery mechanism][discovery] that relies on the DNS to implement a federated namespace that facilitates distributed retrieval.

Hosting ACI images is as simple as including some templated HTML `meta` tags that point to the image artifacts in a web page living under the DNS name that corresponds to the image to host.

This means that, to host ACI images, you only need a web server serving an HTML page with the right `meta` tags and somewhere to host the artifacts.

## Example

For the `coreos.com/etcd` image, you can find in the source the following `meta` tags:

```
<meta name="ac-discovery" content="coreos.com/etcd https://github.com/coreos/etcd/releases/download/{version}/etcd-{version}-{os}-{arch}.{ext}">
<meta name="ac-discovery-pubkeys" content="coreos.com/etcd https://coreos.com/dist/pubkeys/aci-pubkeys.gpg">
<meta name="ac-discovery-pubkeys" content="coreos.com/etcd https://coreos.com/dist/pubkeys/app-signing-pubkey.gpg">
```

When a user tries to fetch this image with the command:

```
$ rkt fetch coreos.com/etcd:v2.0.10
```

These are the steps rkt will do:

* Go to `coreos.com/etcd` and look for `ac-discovery-pubkeys` tags where the `content` prefix matches `coreos.com/etcd`, fetch the public keys, and prompt the user to trust them if they're not trusted already.
* Look for an `ac-discovery` tag with matching `content`.
The first line of our example tags matches that so, to fetch the artifacts, rkt will perform a simple template substitution:
 * It will substitute `{version}` with `v2.0.10`
 * It will substitute `{os}` with the current OS (for example, `linux`)
 * It will substitute `{arch}` with the current architecture (for example, `amd64`).
 * It will substitute `{ext}` with `aci` for the actual image and `aci.asc` for the image signature.
* Fetch the image and signature from the resulting URL and verify that the image has a valid and trusted signature.

## ACI server example

Let's use Python's built-in HTTP server to host an example ACI.

We create an minimal `index.html` file with an `ac-discovery` tag:

```html
<html>
    <head>
        <meta name="ac-discovery" content="localhost/postgres http://localhost/postgres-{version}-{os}-{arch}.{ext}">
    </head>
</html>
```

Put the ACI file in the same directory and start the server on port 80:

```bash
$ cd /tmp/acis
$ ls
index.html  postgres-latest-linux-amd64.aci
$ sudo python3 -m http.server 80
Serving HTTP on 0.0.0.0 port 80 (http://0.0.0.0:80/) ...
```

Now we can fetch the image.
To make things simple, we'll disable image verification and use HTTP instead of HTTPs:


```bash
$ sudo rkt --insecure-options=http,image fetch localhost/postgres
Downloading ACI: [=============================================] 7.46 MB/7.46 MB
Downloading ACI: [=============================================] 2.65 MB/2.65 MB
sha512-f5d991eed255cd081b4ea6e1b378eab4
```

[ACI]: https://github.com/appc/spec/blob/v0.8.1/spec/aci.md
[application-container]: https://github.com/appc/spec#what-is-an-application-container
[discovery]: https://github.com/appc/spec/blob/v0.8.1/spec/discovery.md
