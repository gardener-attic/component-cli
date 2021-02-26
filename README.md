# Component CLI

This repository contains a cli tool to upload component descriptors as defined by the [CTF](https://gardener.github.io/component-spec/).

See the cli help (`component-cli --help`) or the [generated documentation](./docs/reference/components-cli.md) for detailed documentation details.

#### Usage

Install the cli tool locally by running `make install` or `go install ./cmd/...`.
The cli is installed into `$GOBIN/component-cli` and can be used accordingly.

In addition, an OCI Image is provided `docker run eu.gcr.io/gardener-project/component/cli --help`
