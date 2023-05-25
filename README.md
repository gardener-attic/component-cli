# Deprecation Note

`component-cli` is deprecated and will see no further maintenance. It is superseded by
[ocm-cli](https://github.com/open-component-model/ocm).


# Component CLI
This repository contains a cli tool to work with component archives as defined by the [Component Spec](https://gardener.github.io/component-spec/).

## Documentation
The written documentation can be found in [docs/README.md](./docs/README.md). For help on the cli commands, see the cli help (`component-cli --help`) or the [generated documentation](./docs/reference/component-cli.md).

## Installation

### Download Binaries
Download packaged binaries from the [releases page](https://github.com/gardener/component-cli/releases/latest).

### Build from Source
Install the cli locally by running `make install` or `go install ./cmd/...` in the repository root directory. The cli is then installed into `$GOBIN/component-cli` and can be used accordingly.

### Docker Image
In addition, a Docker Image is provided under `eu.gcr.io/gardener-project/component/cli`.
