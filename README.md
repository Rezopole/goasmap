# GoASMap
Publish IP(v4/v6) -> ASnum bindings via dns
GoASMap allows to retrieve an AS information by providing a prefix or vise versa via a dns request.

## Installation
`go get github.com/Rezopole/goasmap`

## Usage

### Server
To launch GoASMap a configuration file is needed
`goasmap -f /pth/to/config/file`

### Client
GoASMap uses DNS requests to map IP and ASN
`origin.yourzone` is used to map an IPv4 address or prefix to an ASN
`origin6.yourzone` is used to map an IPv6 address or prefix to an ASN
`yourzone` is used to map an ASN to a list of IPv4 and IPv6 prefix

IPv4 example:
```bash
$ dig +short 1.1.168.192.origin.yourzone TXT
 "65001 | 192.168.1.0/24"
 ```
 
IPv6 example:
```bash
$ dig +short 0.0.0.0.0.0.c.f.origin6.yourzone TXT
 "65002 | fc00::/7"
 ```

As of now the ASN request provides a list of all prefix belonging to said ASN, this shall be changed in future updates.
ASN example:
```bash
$ dig +short AS65003.yourzone TXT
 "192.168.1.0/24"
 ```

## Prerequisites
You need to install Go 1.5 or later.

## Configuration
GoASMap can be configured via a configuration file defined with the `-f` flag.
The sole configuration format as of now is `toml`.

GoASMap uses the same configuration syntax as GoBGP (https://github.com/osrg/gobgp/blob/master/docs/sources/getting-started.md), with added fields for GoASMap.

```toml
[global.config]
as = 64512
router-id = "192.168.255.1"

[[neighbors]]
[neighbors.config]
neighbor-address = "10.0.255.1"
peer-as = 65001
[neighbors.ebgp-multihop.config]
enabled = true
multihop-ttl = 3

[dns]
ip = ""
port = "53"
zone = "/path/to/zone/file.zone"
max = 100

[IPV6]
ip = ["fc00::/7,65003"]
include = ["/custom/iplist/ipv6_custom"]

[IPV4]
ip = ["192.168.1.0/24,65002"]
include = ["/custom/iplist/ipv4_custom"]
```

Custom IPv4 and IPv6 fields can be added with `ip` as key and an array containing a prefix and ASN or any other information you wish to display, separated by a comma as value, or files can be included with `include` as key that has an array of paths to said files containing the same format.

```bash
$ cat /custom/iplist/ipv4_custom
[IPV4]
ip = ["0.0.0.0/8,LOCAL IDENTIFICATION",
"10.0.0.0/8,RCF1918",
"192.168.0.0/16,RFC1918",
"42.42.42.0/24,65004"]
```


## Dependencies
GoASMap uses govendor to manage dependencies with third party packages.
To download all tested versions of all packages used by GoASMap use the following command:
```bash
govendor sync
```
