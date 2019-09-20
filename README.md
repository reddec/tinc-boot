# Overview

Idea to create a easy-to-use wrapper over [tinc vpn](https://www.tinc-vpn.org).

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


## Installation

* (recommended) look at releases page and download
* build from source `go get -v github.com/reddec/tinc-boot/cmd/...`

### Build requirements

* go 1.13+

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
sudo tinc-boot bootnode --service --dir /etc/tinc/dnet --token <SECRETTOKEN>
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

TBD

# TODO

* generate script with token to redistribute all-in-one to end-users
