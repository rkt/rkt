# go-acl

Golang POSIX.1e ACL bindings.
Essentially bindings to /usr/include/sys/acl.h
## notes

### mac os x
Mac OS X does not seem to support basic POSIX1.e ACLs. They do
provide the POSIX API for NFSv4 ACLs. It would be nice for this
package to also support NFSv4 ACLs.

### freebsd
By default, FreeBSD does not enable POSIX1.e ACLs on the root
partition. To enable them, reboot into single-user mode and execute:

    $ tunefs -a enable
    $ reboot

Source: https://www.freebsd.org/doc/handbook/fs-acl.html

## info

The IEEE POSIX.1e specification describes five security extensions to the base
POSIX.1 API: Access Control Lists (ACLs), Auditing, Capabilities,
Mandatory Access Control, and Information Flow Labels.
The specificaiton was abandoned before finalization, however most
UNIX-like operating systems have some form of ACL implementation.

Source: http://www.gsp.com/cgi-bin/man.cgi?section=3&topic=posix1e
