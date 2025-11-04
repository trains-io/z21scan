# The z21 Scanner

A command line utility to scan a local network for reachable z21 devices.

## Installation

Releases are [published to GitHub](https://github.com/trains-io/z21scan/releases) where Zip, RPMs and DEBs for various operating systems can be found.

### Installation via go install

The z21scan tool can be installed directly via `go install`. To install the latest version:

```sh
go install github.com/trains-io/z21scan@latest
```

To install a specific release:

```sh
go install github.com/trains-io/z21scan@v0.0.2
```

## Local Network Scan

The scanner can probe all hosts in a given network range, or all devices connected to a specific network interface.

### Example 1 — Scan by network address

```sh
z21scan 192.168.2.0/24
```

This will send a UDP probe to each IP address in the specified range and report any reachable z21 device.

### Example 2 — Scan by network interface

```sh
z21scan eth0
```

This will automatically determine the IP and mask of the given interface and scan all hosts in that subnet.

## Output Formats

The output format can be selected using the `--output` (or `-o`) flag.

Available modes:

| Mode      | Description                                          |
| --------- | ---------------------------------------------------- |
| `short`   | Prints only the IP addresses of reachable devices    |
| `normal`  | Shows IP, port, and device serial                    |
| `verbose` | Same as normal, but includes extra diagnostic output |
| `json`    | Machine-readable JSON array of results               |

### Examples

### Normal Output

```sh
z21scan 192.168.2.0/24 -o normal
```

Output

```sh
Found 2 Z21 device(s):
  192.168.2.6      port=21105 serial=265070
  192.168.2.7      port=21105 serial=265071
```

### JSON Output

```sh
z21scan 192.168.2.0/24 -o json
```

Output

```sh
[
  {"ip":"192.168.2.6","port":21105,"reachable":true,"serial":"265070"},
  {"ip":"192.168.2.7","port":21105,"reachable":true,"serial":"265071"}
]
```

## License

This project is licensed under the MIT License.

## Contributing

Contributions, bug reports, and feature requests are welcome!
Simply open an issue or submit a pull request.
