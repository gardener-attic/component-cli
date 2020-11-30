# Component CLI

This repository contains a cli tool to upload component descriptors as defined by the [CTF](https://gardener.github.io/component-spec/).

#### Usage

Install the cli tool locally by running `make install` or `go install ./cmd/...`.
The cli is installed into `$GOBIN/component-cli` and can be used accordingly.

In addition, an OCI Image is provided `docker run eu.gcr.io/gardener-project/component/cli --help`
