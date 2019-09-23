# Overview

**Tinc-boot** - is a all-in-one tool with zero dependency (except `tinc` of course), that aims to achieve:

1. one-line node initialization
2. automatic keys distribution
3. simplified procedure to add new node to existent net

* **Author**: Baryshnikov Aleksandr (reddec) <owner@reddec.net>
* **License**: Mozilla Public License Version 2.0

# Usage

`tinc-boot [OPTIONS] <command>`


**Help Options:**

*  `-h, --help`  Show this help message

Available commands:

* `gen`       - Generate new tinc node over bootnode  
* `bootnode`  - Serve as a boot node
* `monitor`   - Run as a daemon for watching new subnet and provide own host key (tinc-up)
* `watch`     - Add new subnet to watch daemon to get it host file (subnet-up)
* `forget`    - Forget subnet and stop watching it (subnet-down)
* `kill`      - Kill monitor daemon (tinc-down)

## gen

Generate new tinc node using bootnode or standalone.  

**Usage:**

`tinc-boot [OPTIONS] gen [gen-OPTIONS] [URLs...]`

**Help Options:**

* `-h, --help` -            Show this help message

**Options:**

* `--network=`    - Network name (default: dnet) [$NETWORK]
* `--name=`       - Self node name (trimmed hostname will be used if empty) [$NAME]
* `--dir=`        - Configuration directory (default: /etc/tinc) [$DIR]
* `-t, --token=`  - Authorization token (used as a encryption key) [$TOKEN]
* `--prefix=`     - Address prefix (left segments will be randomly auto generated) (default: 172.173) [$PREFIX]
* `--mask=`       - Network mask (default: 16) [$MASK]
* `--timeout=`    - Boot node request timeout (default: 15s) [$TIMEOUT]
* `--bin=`        - tinc-boot location (default: /usr/local/bin/tinc-boot) [$BIN]
* `--no-bin-copy` - Disable copy tinc-boot binary [$NO_BIN_COPY]
* `--port=`       - Node port (first available will be got if not set) [$PORT]
* `-a, --public=` - Public addresses that could be used for incoming connections [$PUBLIC]
* `--standalone`  - Do not use bootnodes (usefull for very-very first initialization) [$STANDALONE]

**Arguments:**
  
* `URLs`          - boot node urls


**Notes:**

* In case default parameters are changed, the same parameters should passed everywhere.
* Name of target configuration directory is a combination of configuration directory and normalized network name. 
For the default settings - `/etc/tinc/dnet`.
* Token flag (`-t, --token`) is optional only for standalone (`--standalone`) nodes.
* VPN address (`--prefix`) finally should be 4 segments (x.y.z.t). 
If provided less then 4, left over parts will be randomly auto-generated.
* Mask (`--mask`) is used to explain Tinc which addresses should be routed inside network.
* By default `tinc-boot` will try to copy itself to the `--bin` location (if not exists). To prevent it - use `--no-bin-copy`.
* Public address (`-a`) could be defined many times.
* URLs should be full url (ex: `http://example.com` and `https://example.com`) or just address (ex: 1.2.3.4:8665). 

## bootnode

Serve as a boot node. Traffic between client and server encrypted by xchacha20poly1305 over HTTP.

**Usage:**

`tinc-boot [OPTIONS] bootnode [bootnode-OPTIONS]`
  
**Help Options:**

* `-h, --help` -            Show this help message

**Options:**
          
* `--name=`     - Self node name (hostname by default) [$NAME]
* `--dir=`      - Configuration directory (including net) (default: /etc/tinc/dnet) [$DIR]
* `--binding=`  - Public binding address (default: :8655) [$BINDING]
* `--token=`    - Authorization token (used as a encryption key) [$TOKEN]
* `--service`   - Generate service file to /etc/systemd/system/tinc-boot-{net}.service [$SERVICE]
* `--tls-key=`  - Path to private TLS key [$TLS_KEY]
* `--tls-cert=` - Path to public TLS certificate [$TLS_CERT]

**Notes:**

* In case of custom configuration, flags `--name` and `--dir` should be adjusted.
* Token (`--token`) may contains any value. It will be normalized internally. Token MUST be same on client side.

## monitor

Run as a daemon for watching new subnet and provide own host key (used in script `tinc-up`). 
The command specially designed (including environment variables) to be used in a Tinc script. 
There is **no need to launch the command manually**. 

**Usage:**

`tinc-boot [OPTIONS] monitor [monitor-OPTIONS]`

**Help Options:**

* `-h, --help` -            Show this help message

**Options:**

* `--iface=`    - Interface to bind [$INTERFACE]
* `--dir=`      - Configuration directory (default: .) [$DIR]
* `--name=`     - Self node name [$NAME]
* `--port=`     - Port to bind (should same for all hosts) (default: 1655) [$PORT]
* `--timeout=`  - Attempt timeout (default: 30s) [$TIMEOUT]
* `--interval=` - Retry interval (default: 10s) [$INTERVAL]
* `--reindex=`  - Reindex interval (default: 1m) [$REINDEX]

**Notes:**

* Monitor serves host file over http interface inside VPN.
* Monitor tries to get host file from other peers with defined timeout (`--timeout`) for requests every `--interval`.
* Monitor scans host files for a new public nodes after a new peer or after `--reindex` interval.

## watch

Add new subnet to watch daemon to get it host file (used in script `subnet-up`).
The command specially designed (including environment variables) to be used in a Tinc script. 
There is **no need to launch the command manually**.

**Usage:**

`tinc-boot [OPTIONS] watch [watch-OPTIONS]`

**Help Options:**

* `-h, --help` -            Show this help message

**Options:**

* `--iface=`  - RPC interface [$INTERFACE]
* `--port=`   - RPC port (default: 1655) [$PORT]
* `--subnet=` - Subnet address to watch [$SUBNET]
* `--node=`   - Subnet owner name [$NODE]


## forget

Forget subnet and stop watching it (used in script `subnet-down`).
The command specially designed (including environment variables) to be used in a Tinc script. 
There is **no need to launch the command manually**.

**Usage:**

`tinc-boot [OPTIONS] forget [forget-OPTIONS]`

**Help Options:**

* `-h, --help` -            Show this help message

**Options:**

* `--iface=`  - RPC interface [$INTERFACE]
* `--port=`   - RPC port (default: 1655) [$PORT]
* `--subnet=` - Subnet address to forget [$SUBNET]
* `--node=`   - Subnet owner name [$NODE]

## kill


**Usage:**

`tinc-boot [OPTIONS] kill [kill-OPTIONS]`

**Help Options:**

* `-h, --help` -            Show this help message


**Options:**
          
* `--iface=` - RPC interface [$INTERFACE]
* `--port=`  - RPC port (default: 1655) [$PORT]

