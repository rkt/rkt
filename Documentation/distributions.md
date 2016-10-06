# Installing rkt on popular Linux distributions

- [Arch](#arch)
- [CentOS](#centos)
- [CoreOS](#coreos)
- [Debian](#debian)
- [Fedora](#fedora)
- [NixOS](#nixos)
- [openSUSE](#opensuse)
- [Ubuntu](#ubuntu)
- [Void](#void)

## Arch

rkt is available in the [Community Repository](https://www.archlinux.org/packages/community/x86_64/rkt/) and can be installed using pacman:
```
sudo pacman -S rkt
```

## CentOS

rkt is available in the [CentOS Community Build Service](https://cbs.centos.org/koji/packageinfo?packageID=4464) for CentOS 7.
However, this is [not yet ready for production use](https://github.com/coreos/rkt/issues/1305) due to pending systemd upgrade issues.

## CoreOS

rkt is an integral part of CoreOS, installed with the operating system.
The [CoreOS releases page](https://coreos.com/releases/) lists the version of rkt available in each CoreOS release channel.

If the version of rkt included in CoreOS is too old, it's fairly trivial to fetch the desired version [via a systemd unit](install-rkt-in-coreos.md).

## Debian

rkt is currently packaged in Debian sid (unstable) available at https://packages.debian.org/sid/utils/rkt:

```
sudo apt-get install rkt
```

Note that due to an outstanding bug (https://bugs.debian.org/cgi-bin/bugreport.cgi?bug=823322) one has to use the "coreos" stage1 image:

```
sudo rkt run --insecure-options=image --stage1-name=coreos.com/rkt/stage1-coreos:1.5.1 docker://nginx
```

If you prefer to install the newest version of rkt follow the instructions in the Ubuntu section below.

## Fedora

rkt is packaged in the development version of Fedora, [Rawhide](https://fedoraproject.org/wiki/Releases/Rawhide):
```
sudo dnf install rkt
```

Until the rkt package makes its way into the general Fedora releases, [download the latest rkt directly from the project](https://github.com/coreos/rkt/releases).

rkt's entry in the [Fedora package database](https://admin.fedoraproject.org/pkgdb/package/rpms/rkt/) tracks packaging work for this distribution.

#### Caveat: SELinux

rkt can integrate with SELinux on Fedora but in a limited way.
This has the following caveats:
- running as systemd service restricted (see [#2322](https://github.com/coreos/rkt/issues/2322))
- access to host volumes restricted (see [#2325](https://github.com/coreos/rkt/issues/2325))
- socket activation restricted (see [#2326](https://github.com/coreos/rkt/issues/2326))
- metadata service restricted (see [#1978](https://github.com/coreos/rkt/issues/1978))

As a workaround, SELinux can be temporarily disabled:
```
sudo setenforce Permissive
```
Or permanently disabled by editing `/etc/selinux/config`:
```
SELINUX=permissive
```

#### Caveat: firewalld

Fedora uses [firewalld](https://fedoraproject.org/wiki/FirewallD) to dynamically define firewall zones.
rkt is [not yet fully integrated with firewalld](https://github.com/coreos/rkt/issues/2206).
The default firewalld rules may interfere with the network connectivity of rkt pods.
To work around this, add a firewalld rule to allow pod traffic:
```
sudo firewall-cmd --add-source=172.16.28.0/24 --zone=trusted
```

172.16.28.0/24 is the subnet of the [default pod network](https://github.com/coreos/rkt/blob/master/Documentation/networking/overview.md#the-default-network). The command must be adapted when rkt is configured to use a [different network](https://github.com/coreos/rkt/blob/master/Documentation/networking/overview.md#setting-up-additional-networks) with a different subnet.

## NixOS

rkt can be installed on NixOS using the following command:

```
nix-env -iA rkt
```

The source for the rkt.nix expression can be found on [GitHub](https://github.com/NixOS/nixpkgs/blob/master/pkgs/applications/virtualization/rkt/default.nix)


## openSUSE

rkt is available in the [Virtualization:containers](https://build.opensuse.org/package/show/Virtualization:containers/rkt) project on openSUSE Build Service.
Before installing, the appropriate repository needs to be added (usually Tumbleweed or Leap):

```
sudo zypper ar -f obs://Virtualization:containers/openSUSE_Tumbleweed/ virtualization_containers
sudo zypper ar -f obs://Virtualization:containers/openSUSE_Leap_42.1/ virtualization_containers
```

Install rkt using zypper:

```
sudo zypper in rkt
```

## Ubuntu

rkt is not packaged currently in Ubuntu. If you want the newest version of rkt the easiest method to install it is using the `install-rkt.sh` script. This script can be found in the *scripts* directory of the [rkt Github repository](https://github.com/coreos/rkt).

Either clone the repository and run the script:

```
git clone https://github.com/coreos/rkt
cd rkt
sudo ./scripts/install-rkt.sh
```

Or just get the script:

```
wget https://raw.githubusercontent.com/coreos/rkt/master/scripts/install-rkt.sh
chmod +x install-rkt.sh
sudo ./install-rkt.sh
```

The above script will install the "gnupg2", and "checkinstall" packages, download rkt, verify it, and finally invoke "checkinstall" to create a deb package and install rkt. To uninstall rkt, execute:

```
sudo apt-get remove rkt
```

## Void

rkt is available in the [official binary packages](http://www.voidlinux.eu/packages/) for the Void Linux distribution.
The source for these packages is hosted on [GitHub](https://github.com/voidlinux/void-packages/tree/master/srcpkgs/rkt).
