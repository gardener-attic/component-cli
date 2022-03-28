# Documentation
This document gives an overview and simple examples about the usage of the component-cli.
For help on the cli commands, see the cli help (`component-cli --help`) or the [generated documentation](./docs/reference/component-cli.md).

__Index__:

- [Create a Component Archive](#create-a-component-archive)
  - [Create a Component Archive Skeleton](#create-a-component-archive-skeleton)
  - [Add a Resource](#add-a-resource)
  - [Add a local Blob](#add-a-local-file)
  - [Use Templating](#use-templating)
  - [Add a Reference to another Component Archive](#add-a-reference-to-another-component-archive)
- [Working with OCI Registries as Component Repositories](#working-with-oci-registries-as-component-repositories)
  - [Authenticating to private OCI Registries](#authenticating-to-private-oci-registries)
  - [Push a Component Archive](#push-a-component-archive)
  - [Pull a Component Descriptor](#pull-a-component-descriptor)
  - [Copy a Component Archive](#copy-a-component-archive)
- [Signing](#signing)
  - [Sign and Verify with RSA](#sign-and-verify-with-rsa)

## Create a Component Archive
In the beginning a component archive must be created. A component archive is a directory that contains the component descriptor at `/component-descriptor.yaml` and all resource blobs at `/blobs/<blobname>`.

It is possible to create these primitives either by hand, use a third-party templating engine, or use the utility functions that the component-cli provides.

### Create a Component Archive Skeleton
A component archive skeleton can be created by using the `component-cli component-archive create` command.

```shell script
component-cli component-archive create ./examples --component-name "example.com/component/name" --component-version "v0.0.1"
```

This will create a new directory `./examples` which contains a `component-descriptor.yaml` skeleton file.

### Add a Resource
Resources can be added to an existing component descriptor by using the `component-cli component-archive resources add` command.

```shell script
component-cli ca resources add ./examples ./resources.yaml
```

The previous command adds all resources defined in `./resources.yaml` to the `./examples/component-descriptor.yaml` file.

An example for `./resources.yaml` is presented in the following.

```yaml
# resources.yaml

name: 'ubuntu'
version: '18.0.0'
type: 'ociImage'
relation: 'external'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:18.0'
```

The resources file is expected to be in yaml format, and multiple resources can be added by using the yaml multi doc syntax:

```yaml
---
name: 'ubuntu'
version: '18.0.0'
type: 'ociImage'
relation: 'external'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:18.0'
...
---
name: 'ubuntu'
version: '17.0.0'
type: 'ociImage'
relation: 'external'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:17.0'
...
```

### Add a local File
A local blob (any file) can be added to the component descriptor by using the `input` attribute. This will automatically add the file as local artifact to the component descriptor and will configure the access of the resource accordingly.

> Note you can specify that the given blob is automatically gzipped by setting the `compress` attribute.

> When the given input path is a directory, a tar archive is automatically created.

```yaml
# resources.yaml

name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "file"
  path: "./blob.json" # path is relative to the current resource file
  mediaType: "application/json" # optional, will be defaulted to application/octet-stream
```

```json
// blob.json
{
  "key": "value"
}
```

The command

```shell script
component-cli component-archive resources add ./examples ./resources.yaml
```

will add the blob resource to `./examples/component-descriptor.yaml` and will create a blob file in the `./examples/blobs` directory which contains the file contents.

See an example for a local blob resource with a directory and possible options.

```yaml
---
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "file"
  path: "some/path"
...
---
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "dir"
  path: /my/path
  compress: true # defaults to false
  exclude: "*.txt"
...
```

## Add a Reference to another Component Archive
A component reference can be added to an existing component descriptor by using the `component-cli component-archive component-reference` subcommand.

The subcommand offers the possibility to add a component reference either via component reference template in a file or by defining the component-reference via commandline flags.

> (!) Commandline flags overwrite values defined by the file.

A component reference template can be like in the following snippet.

```yaml
# comp-ref.yaml

name: 'component-spec'
componentName: 'github.com/gardener/component-spec'
version: 'v0.0.1'
```

Via the command

```shell script
component-cli ca component-reference add ./examples ./comp-ref.yaml
```

the component reference defined in `comp-ref.yaml` is added to `./examples/component-descriptor.yaml`.

## Use Templating
The previous commands introduced how resources, sources and component-references can be dynamically added to a component descriptor via the component-cli. Often these definitions need to be templated with the current build values like the version.

One solution for that issue is to do the templating yourself, with your preferred templating engine.
This approach is also recommended when you need a more advanced templating features like loops.

In most use cases, a simple variable substitution is enough to meet the requirements. Therefore, the component-cli offers the possibility to use simple variable expansion in the templates.

For example if a resources need to be templates with a new version, the resource definition would be defined as follows:

```yaml
# resources.yaml

name: 'ubuntu'
version: '${VERSION}'
type: 'ociImage'
relation: 'external'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:${VERSION}'
```

With the command `component-cli component-archive resources add [path to component archive] ./resources.yaml -- VARIABLE=v0.0.2` it is now possible to define key-value pairs for the substitution.

## Working with OCI Registries as Component Repositories
Component archives are typically stored in OCI registries. The component-cli provides commands to interact with these stored components and also resources that are stored in OCI registries (e.g. Docker Images).

### Authenticating to private OCI Registries
The credentials for accessing private registries are handled via plain Docker CLI mechanisms. Either use the `docker login` command, or edit the Docker config file on your local machine manually.

Specialties regarding credential handling in the component-cli:

1) The component-cli uses credential entries from the `auths` section in preference over `credHelpers` for the same registry URL.

```json
{
	"auths": {
		"eu.gcr.io": {}
	},
	"credHelpers": {
		"eu.gcr.io": "gcloud",
	},
	...
}
```

With the above Docker config, the component-cli will use the set of credentials from the `auths` section when interacting with the `eu.gcr.io` registry.

2) The component-cli supports providing credentials for subpaths of the same host (not supported by Docker CLI).

```json
{
	"auths": {
		"eu.gcr.io/my-project": {},
    "eu.gcr.io": {}
	},
	...
}
```

With the above Docker config, the component-cli will use the more specific set of credentials for all artifacts under the path `eu.gcr.io/my-project`.

### Push a Component Archive
A component archive can be pushed to an oci registry by using the `component-cli component-archive remote push` command. 

```shell script
component-cli ca remote push [path to component archive]
```

The command takes 1 argument which is the path to the component archive.

### Pull a Component Descriptor
The component descriptor of a component archive can be pulled from an oci registry by using the `component-cli component-archive remote get` command.

```shell script
component-cli component-archive remote get BASE_URL COMPONENT_NAME VERSION
```

### Copy a Component Archive
The component descriptor of a component archive can be copied between different oci registries by using the `component-cli component-archive remote copy` command.

```shell script
component-cli component-archive remote copy github.com/test-component v0.1.0 --from eu.gcr.io/source --to eu.gcr.io/target
```

The previous command will copy the defined component and recursively also all referenced components from `eu.gcr.io/source` to `eu.gcr.io/target`. Local blob resources, as they are part of the component archives themselves, are also copied to the new target location. 

By passing the cli flag `--copy-by-value`, additionally all resources wit `accessType: ociRegistry` (e.g. Docker Images) will be copied to the target location. Therefore, if your component archives only describe local blobs and oci artifacts as resources, the whole application can be copied in a self contained way between different registries. 

## Signing
> (!) This is currently an experimental feature

To verify the integrity of component descriptors, component-cli provides signing functionality. All signing related commands are placed under the `component-cli component-archive signature` command. The most important subcommands are `sign` and `verify`, which again have subcommands to sign and verify component descriptors using different algorithms. For detailed information on how a component descriptor is signed and verified, visit the [Component Spec](https://gardener.github.io/component-spec/).

### Sign and Verify with RSA
One possible algorithm to sign and verify component descriptors is RSA. RSA requires a private/public keypair for signing and verification, which can be generated with `openssl`.

#### Sign
The command `component-cli component-archive signature sign rsa` signs a component descriptor with RSA. 

```shell script
component-cli ca signature sign rsa eu.gcr.io/unsigned github.com/test-component v0.1.0 --upload-base-url eu.gcr.io/signed --recursive --signature-name test-signature --private-key ./private-key.pem
```

The previous command would recursively download the targeted component descriptor and all referenced components, sign them, and re-upload them to the new component repository defined by the parameter `--upload-base-url`. To overwrite the existing component descriptors with the signed ones, set `--upload-base-url` to the unsigned component repository URL and set the `--force` flag.

In the signed component descriptor a new signature has been appended to `signatures`.

```yaml
meta:
  schemaVersion: v2
component:
  ...
signatures:
- name: test-signature
  digest:
    hashAlgorithm: sha256
    normalisationAlgorithm: jsonNormalisation/V1
    value: c349a4aee7061e6553f0a5e9df840328a2018168c6b2a1475d10955e2114afb3
  signature:
    algorithm: RSASSA-PKCS1-V1_5
    mediaType: application/vnd.ocm.signature.rsa
    value: b0f556e19d78e5b9bfbb1ee9145920d477ab12ee2f556e78b309613131295af78d83ef37f4b24ce7e5f814f04bb564a6dc9b697259ed7462a22a46aaf56bad90ca13e7f94a56018744aef21137cd45e8e83311510bea2bc8b46786d2c0ba0c2c00d6e8038ff9a869bf3bab06167cd5b749eba9f8676c47ac767b5b886ee3765bf31bed4ccdafe63ecc492a17862d056597f3831aa7fa78ca4eab9707f789afe147a34486d86df06635b654af29dfb430991bb61aa7a2a995fa1ea0c73fc1af7c4f8bd390dd4345eec002974dc251df962766dd97dafc77b9d7d8b10b3bedd4fb4939e38f092ed4c39fc4bb26a06e750ed661a4aa883060de794b3a5d8dff0f9b
```

The RSA private key that is used for signing is defined by the parameter `--private-key`. It must point to a file which contains the private key in PEM format. 

The parameter `--signature-name` defines the symbolic name under which the signature is written to the component descriptor. It  allows to differentiate multiple signatures.

#### Verify
The command `component-cli component-archive signature verify rsa` allows you to verify an existing RSA signature inside of a component descriptor. 

```shell script
component-cli ca signature verify rsa eu.gcr.io/signed github.com/test-component v0.1.0 --signature-name test-signature --public-key ./public-key.pem
```

It will recursively fetch the resources and calculate the digests for the component that should be verified. In the end is checked that the calculated digest matches the signed digest in the signature that is selected with the parameter `--signature-name`. 

The RSA public key that is used for verifying is defined by the parameter `--public-key`. It must point to a file which contains the public key in PEM format.
