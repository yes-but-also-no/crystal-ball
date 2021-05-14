# Orakuru's crystal-ball

Node for providing data into Orakuru network.

## Configuration

Crystal-ball uses environment variables and configuration files for configuration.
List of environment variables with their meaning:

* `CB_CONFIG_DIR` - path to `etc` directory containing `web3.yml` and `requests.yml`. Default is `etc/`
* `CB_LOG_LEVEL` - log level of the node, from `trace` to `panic`. Default is `info`
* `CB_PRETTY_LOG` - if set to `true`, outputs logs in a pretty format, otherwise uses JSON. Default is `true`
* `MONITORING_HOST` - `host:port` on which Prometheus monitoring will be exposed. Default is `:9000`

Explanation of specific configuration files is provided as comments in examples (`etc/` in this repo).

## Installation

Recommended way of running a node is through Docker. You'll need to create configuration files first. An example command for starting node:

```shell
$ docker run -v /absolute/path/to/etc:/go/src/app/etc -d \ # Path to configuration
             -e CB_LOG_LEVEL=trace \ # Enable debug logging
             --name "crystal-ball" \ # Set name for container
             ghcr.io/orakurudata/crystal-ball:v0.1.1
```

**Notice:** before running this command, you need to create a configuration directory somewhere on your machine, copy [Web3](etc/web3.yml) and [Requests](etc/requests.yml) configuration files to that directory, and modify them (change web3 endpoint, private key). After that, replace `/absolute/path/to/etc` with an absolute path to your newly created configuration directory. 
This step is **REQUIRED**.

You can also build your version of node from scratch. You'll need to install Go 1.16 or higher, GCC, G++, linux-headers, and git.
As of now, we officially only support Linux, but node probably will build on Windows and macOS.

To build node from scratch, follow these steps:

```shell
$ git clone https://github.com/orakurudata/crystal-ball
$ cd crystal-ball
$ go get ./...
$ go install ./cmd/crystal-ball
```

This will build and install `crystal-ball` executable to `$GOPATH/bin`.