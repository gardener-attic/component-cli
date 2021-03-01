# Component CLI

This repository contains a cli tool to upload component descriptors as defined by the [CTF](https://gardener.github.io/component-spec/).

See the cli help (`component-cli --help`) or the [generated documentation](./docs/reference/components-cli.md) for detailed documentation details.

#### Usage

1. Get the latest release with:
   ```
   curl -L https://github.com/gardener/component-cli/releases/download/$(curl -s https://api.github.com/repos/gardener/component-cli/releases/latest | jq -r '.tag_name')/componentcli-linux-amd64.gz | gzip -d > ./component-cli
   ```
   > This example downloads the linux amd64 binary. Make sure to replace the system information with yours. 

   To download a specific version, replace `$(curl -s https://api.github.com/repos/gardener/component-cli/releases/latest | jq -r '.tag_name')` with the specific version.
   ```
   curl -L https://github.com/gardener/component-cli/releases/download/v0.9.0/componentcli-linux-amd64.gz | gzip -d > ./component-cli
   ```

2. Make the binary executable and install the binary
   ```
   chmod +x ./component-cli
   mv ./component-cli /usr/local/bin/component-cli
   ```
3. Ensure the cli is correctly installed
   ```
   component-cli version
   ```


##### Build from Source

Install the cli tool locally by running `make install` or `go install ./cmd/...`.
The cli is installed into `$GOBIN/component-cli` and can be used accordingly.

In addition, an OCI Image is provided `docker run eu.gcr.io/gardener-project/component/cli --help`
