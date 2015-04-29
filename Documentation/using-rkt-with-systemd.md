# Using rkt with systemd

**work in progress**

This document describes how to use rkt with systemd.

## Overview

rkt is designed and intended to be used with init systems like [systemd](http://www.freedesktop.org/wiki/Software/systemd/).
Since rkt consists only of a simple CLI that directly executes processes and writes to stdout/stderr (i.e. it does not daemonize), the lifecycle of rkt pods can be directly managed by systemd.
Consequently, standard systemd idioms like `systemctl start` and `systemctl stop` work out of the box.

## Example unit file

The following is a simple example of a unit file using rkt to run an nginx image:
```
[Unit]
Description=nginx

[Service]
ExecStart=/usr/bin/rkt --insecure-skip-verify run docker://nginx
Restart=always
```

This unit can now be managed using the standard `systemctl` commands:
```
systemctl start nginx.service
systemctl stop nginx.service
systemctl restart nginx.service
```
