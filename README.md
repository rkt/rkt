# Rocket - App Container Reference Implementation

Rocket is a reference implementation of the app container specification. The
goal of rocket is to be a composable and minimal implementation of the spec.
Other implementations are possible - for example, with a focus on tighter
integration to particular projects, or an emphasis on speed.

## Stages

Execution with Rocket is divided into a number of distinct stages. The
motivation for this is to separate the concerns of initial filesystem setup,
execution environment, and finally the execution of the apps themselves.

### Stage 0

The first step of the process, stage 0, is the actual `rkt` binary itself. This
binary is in charge of doing a number of initial preparatory tasks:
- Generating a Container Unique ID (UID)
- Generating a Container Runtime Manifest
- Creating a filesystem for the container
- Setting up stage 1 and stage 2 directories in the filesystem
- Copying the stage1 binary into the container filesystem
- Fetching the specified TAF
- Unpacking the TAF and copying each app into the stage2 directories

Given a run command such as:

```
rkt run --volume bind:/opt/tenant1/database \
	sha1-8a30f14877cd8065939e3912542a17d1a5fd9b4c \
	example.com/data-downloader-1.0.0 \
	example.com/ourapp-1.0.0 \
	example.com/logbackup-1.0.0
```

a container manifest compliant with the ACE spec will be generated, and the
filesystem created by stage0 should be:

```
/container
/stage1
/stage1/init
/stage1/opt
/stage1/opt/stage2/example_2ecom_2fourapp_2d1_2e0_2e0
/stage1/opt/stage2/example_2ecom_2fdata_2ddownloader_2d1_2e0_2e0
/stage1/opt/stage2/example_2ecom_2flogbackup_2d1_2e0_2e0
```

where:
- `container` is the container manifest file
- `stage1` is a copy of the stage1 filesystem that is safe for read/write
- `stage1/init` is the actual stage1 binary to be executed
- `stage1/opt/stage2` are copies of the applications from the RAFs

At this point the stage0 execs `/stage1/init` with the current working
directory set to the root of the new filesystem.

### Stage 1

The next stage is a binary that the user trusts to set up cgroups, execute
processes, and other operations as root. This stage has the responsibility to
take the execution group filesystem that was created by stage 0 and create the
necessary cgroups, namespaces and mounts to launch the execution group:

- Generate systemd unit files from the Application and Container Manifests
  (containing, respectively, the exec specifications of each container and the
  ordering given by the user)
- Set up any external volumes (undefined at this point)
- nspawn attaching to the bridge and launch the execution group systemd
- Launch the root systemd
- Have the root systemd

This process is slightly different for the qemu-kvm stage1 but a similar
workflow starting at `exec()`'ing kvm instead of an nspawn.

### Stage 2

The final stage is executing the actual application. The responsibilities of
the stage2 include:

- Launch the init process described in the Application Manifest

