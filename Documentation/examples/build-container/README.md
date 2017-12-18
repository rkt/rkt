# Build container examples

This directory includes two examples of scripts that build ACI images.
One is a generic [PostgreSQL][postgresql] image and the other is an example [Play Framework][play-framework] app that connects to a PostgreSQL database.
The example Play Framework app expects a PostgreSQL database, so both images can be used in a single pod to make the example run.

## PostgreSQL

The PostgreSQL image is based on [Alpine Linux][alpine-linux] and defines two mounts, one for a PostgreSQL data directory that will be used if present, and another for an SQL script that can be used to initialize the database.
By default, if no data is found in the PostgreSQL data directory, a database named `rkt` will be created with user and password set to `rkt` too.
Then, the initialization SQL script will run.
If a PostgreSQL data directory is found in the corresponding mount, it will be used instead.

## Example Play Framework app

This is a very simple [example Play Framework app][play-framework-example] that just prints the number of times a page is retrieved.
This value is stored in a PostgreSQL database.

## Build the images

You can build the images by running their build scripts.
They use [containers/build][containers-build] so you'll need to get the [latest release][build-release].
Assuming a privileged shell is running in this directory.

```
# (cd postgres && ./build-postgres.sh)
[...]
# (cd play-example && ./build-play-example.sh)
[...]
```

Once they're done building you should have an ACI image on each directory:

```
# ls */*.aci
play-example/play-latest-linux-amd64.aci  postgres/postgres-latest-linux-amd64.aci
```

## Run the example

You simply need to run rkt with a volume pointing to the play example SQL customization script.
Assuming a privileged shell is running in this directory:

```
# rkt --insecure-options=image \
      run \
      --volume custom-sql,kind=host,source=$PWD/play-example/custom.sql \
      postgres/postgres-latest-linux-amd64.aci \
      play-example/play-latest-linux-amd64.aci
```

After it is running and initialized, you need to retrieve the pod IP address and you can access it on port 9000:

```
# rkt list
UUID		APP		IMAGE NAME			STATE	CREATED		STARTED		NETWORKS
a94b94eb	postgres	example.com/postgres		running	11 seconds ago	11 seconds ago	default:ip4=172.16.28.26
		play-example	example.com/play-example:17.10						
# curl 172.16.28.26:9000
[...]
This page has been retrieved 0 times.
[...]
# curl 172.16.28.26:9000
[...]
This page has been retrieved 1 times.
[...]
```

[postgresql]: https://www.postgresql.org
[play-framework]: https://www.playframework.com
[alpine-linux]: https://alpinelinux.org
[play-framework-example]: https://github.com/ics-software-engineering/play-example-postgresql
[containers-build]: https://github.com/containers/build
[build-release]: https://github.com/containers/build/releases
