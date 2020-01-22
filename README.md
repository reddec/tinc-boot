# Tinc-Boot

[![license](https://img.shields.io/github/license/reddec/tinc-boot.svg)](https://github.com/reddec/tinc-boot)
[![](https://godoc.org/github.com/reddec/tinc-boot?status.svg)](http://godoc.org/github.com/reddec/tinc-boot)
[![donate](https://img.shields.io/badge/help_by️-donate❤-ff69b4)](http://reddec.net/about/#donate)

Idea to create a easy-to-use wrapper over [tinc vpn](https://www.tinc-vpn.org).

[skip to installation](#installation)

Tinc VPN - is full-mesh, auto-healing, time-proofed VPN system without single point of failure, with high-throughput and
serious cryptography. 
All nodes in a Tinc network are fully equal. New nodes discovering full topology through any entry point. 
Node may interact with each other even if they don't have direct connections.

Tinc is a great and have a lot of features. It's ideal for a complicated situations (China, Russia and others). 
I really admire the project.

![transit](https://user-images.githubusercontent.com/6597086/65304801-1b4ae480-dbb4-11e9-933f-b890242358ab.png)

**But...** it's pain to configure and maintain.

Pain to create a new node. Pain to add new node to network.

Minimal configuration for a first public node: 

* 2 files (tinc.conf, hostfile), 
* 1 script (tinc-up), 
* 2 directories (net, hosts), 
* 1 command execution (key generation).

(let's not count service initialization and other common stuff)

Second node adds key exchange (+1 operation if we will use `rsync`, or +2 operations if manually).

![second_node](https://user-images.githubusercontent.com/6597086/65304124-72e85080-dbb2-11e9-939f-6359095dbe54.png)

Next new public nodes require increasing number of additional operations (+N operations, where N is a number of public nodes).

![third_node](https://user-images.githubusercontent.com/6597086/65304303-df634f80-dbb2-11e9-8b9a-32bd4c6b9c46.png)


> To be honest, to just to connect to the network an only single key exchange operation required: with any public node. 
> Than tincd will discover all other nodes.
>
> **But** after your node disconnect/reboot and in case of death of your entry node you will be no more able to connect 
> to other alive nodes (because they don't know your key and your node don't know theirs).



**Tinc-boot** - is a all-in-one tool with zero dependency (except `tinc` of course), that aims to achieve:

1. one-line node initialization
2. automatic keys distribution
3. simplified procedure to add new node to existent net

With simple UI (available on your VPN address with port 1655 by default)

![image](https://user-images.githubusercontent.com/6597086/66646721-92c2df80-ec59-11e9-90b3-153b50dd38be.png)

Donating always welcome

* ETH: `0xA4eD4fB5805a023816C9B55C52Ae056898b6BdBC`
* BTC: `bc1qlj4v32rg8w0sgmtk8634uc36evj6jn3d5drnqy`


## Installation

* (recommended) look at  [releases](https://github.com/reddec/tinc_boot/releases) page and download
* one line shell command:
```
curl -L https://github.com/reddec/tinc-boot/releases/latest/download/tinc-boot_linux_amd64.tar.gz | sudo tar -xz -C /usr/local/bin/ tinc-boot
```
* build from source `go get -v github.com/reddec/tinc-boot/cmd/...`
* [Ansible galaxy](https://galaxy.ansible.com/reddec/tinc_boot): `ansible-galaxy install reddec.tinc_boot`

* From bintray repository for most **debian**-based distribution (`trusty`, `xenial`, `bionic`, `buster`, `wheezy`):
```bash
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 379CE192D401AB61
echo "deb https://dl.bintray.com/reddec/debian {distribution} main" | sudo tee -a /etc/apt/sources.list
sudo apt install tinc-boot
```

### Build requirements

* go 1.13+

## Documentation

* Available by `--help` for all commands
* Available in [MANUAL.md](MANUAL.md)

## Runtime requirements

* Linux
* `tincd 1.10.xx`
* `bash`
* (recommended) `systemd`

## Tested operation systems

* Ubuntu 18.04 x64
* Archlinux (Q1 2019) x64
* Manjaro (Q1 2019) x64

Should work on all major linux systems, except generated helpers useful only for systemd-based OS. 


# Quick start

Download/build binary to `/usr/local/bin/tinc-boot`.

## First node

```
sudo tinc-boot gen --standalone -a <PUBLIC ADDRESS>
```

and follow recommendations

### Explanation

* `--standalone` means that it's a first node, no need for keys exchange
* `-a <address>` sets public address of node (if exists); could be used several times 

Will generate all required files under `/etc/tinc/dnet`.

## Turn node to boot node

```
sudo tinc-boot bootnode --service --token <SECRETTOKEN>
```

and follow recommendations

### Explanation

* `--service` generates systemd file to `/etc/systemd/system/tinc-boot-{net}.service`
* `--dir` location of tinc configuration
* `--token` set's authorization token that will be used by clients 

## Create another node and join to net

```
sudo tinc-boot gen --token <SECRETTOKEN> <PUBLIC ADDRESS>:8655
```

> Don't forget add `-a <NODE ADDRESS>` if applicable

and follow recommendations

# How it works

* [Обзор (RU)](https://habr.com/ru/post/468213)
* [Overview (EN)](https://dev.to/reddec/tinc-boot-full-mesh-vpn-without-pain-3lg9)

![overview](https://user-images.githubusercontent.com/6597086/65752642-ca049d00-e13f-11e9-86ff-05134129eb86.png)

# Windows

Tested only for x64

Requirements:

* Tinc for Windows: [download on official site](https://www.tinc-vpn.org/)
* **Install TAP driver**!:
  * Go to `C:\Program Files(x86)\tinc\tap-win64`
  * As administrator run `addtap.bat`