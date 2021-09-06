# Bobcat Miner 300 Exporter

Prometheus exporter for [Bobcat Miner 300](https://www.bobcatminer.com/) - the [Helium](https://www.helium.com/) Miner.

## Quick Start

This package is available as container:

1. Run Bobcat Exporter, where `http://XXXXXXX` is your Bobcat Miner address.

```bash
docker run -e BOBCAT_EXPORTER_MINER_URI="http://XXXXXXX" pperzyna/bobcat_exporter
```

### Flags

* `bobcat.uri`
  Address of Bobcat Miner. Default is `http://localhost`.

* `bobcat.timeout`
  Timeout request to Bobcat Miner. Default is `5s`.

* `web.listen-address`
  Address to listen on for web interface and telemetry. Default is `:9857`.

* `web.telemetry-path`
  Path under which to expose metrics. Default is `/metrics`.

* `log.level`
  Set logging level: one of `debug`, `info`, `warn`, `error`, `fatal`

* `log.format`
  Set the log output target and format. e.g. `logger:syslog?appname=bob&local=7` or `logger:stdout?json=true`
  Defaults to `logger:stderr`.

### Environment Variables

The following environment variables configure the exporter:

* `BOBCAT_EXPORTER_MINER_URI`
  Address of Bobcat Miner. Default is `http://localhost`.

* `BOBCAT_EXPORTER_MINER_TIMEOUT`
  Timeout reqeust to Bobcat Miner. Default is `30s`.

* `BOBCAT_EXPORTER_WEB_LISTEN_ADDRESS`
  Address to listen on for web interface and telemetry. Default is `:9857`.

* `BOBCAT_EXPORTER_WEB_TELEMETRY_PATH`
  Path under which to expose metrics. Default is `/metrics`.

Settings set by environment variables starting with `BOBCAT_` will be overwritten by the corresponding CLI flag if given.

## Development

The default way to build is:

```bash
go get github.com/pperzyna/bobcat_exporter
cd ${GOPATH-$HOME/go}/src/github.com/pperzyna/bobcat_exporter/
go build -o bobcat_exporter
export BOBCAT_EXPORTER_MINER_URI="http://localhost"
./bobcat_exporter <flags>
```

See [CONTRIBUTING](CONTRIBUTING.md)
