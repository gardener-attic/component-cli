# Documentation

__Index__:
- [Resource](#add-a-resource)
  - [Add a local blob](#add-a-local-file)
- [ComponentReference](#add-a-dependency-to-a-component)
- [Remote](#remote)
  - [Push a Component Descriptor](#push)


### Add a Resource

A resource can be added to an existing component descriptor by using the `resource` subcommand.

The subcommand adds all resources defined in by a file .<br>

```shell script
# define by file
$ cat <<EOF
name: 'ubuntu'
version: 'v0.0.1'
type: 'ociImage'
relation: 'external'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:18.0'
EOF > ./resource.yaml
$ component-cli resources add -r ./resource.yaml
```

The file is expected to be a yaml, and multiple resources can be added by using the yaml multi doc syntax:
```yaml
---
name: 'myconfig'
type: 'json'
relation: 'local'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:18.0'
...
---
name: 'myconfig'
type: 'json'
relation: 'local'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:18.0'
...
```

#### add a local file

A local blob (any file) can be added to the component descriptor by using the `input` attribute.
This will automatically add the file as local artifact to the component descriptor and will configure the access of the resource accordingly.

:warning: Note you can specify that the given blob is automatically gzipped by setting the `compress` attribute.<br>
:warning: When the given input path is a directory, a tar archive is automatically created.

```
$ cat <<EOF
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "file"
  path: "./blob.raw" # path is realtive to the current resource file
EOF > ./resource.yaml

$ cat <<EOF
{
  "key": "value"
}
EOF > ./blob.raw

$ component-cli resources add -r ./resource.yaml
```

See an example with a directory and possible options.
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




### Add a dependency to a component

A component reference can be added to an existing component descriptor by using the `component-reference` subcommand.

The subcommand offers the possibility to add a component-reference either via component-reference template in a file or by defining the component-reference via commandline flags.<br>
:warning: Note that the commandline flags overwrite values defined by the file.

```shell script
# define by file
$ cat <<EOF
name: 'cli'
componentName: 'github.com/gardener/component-spec'
version: 'v0.0.1'
extraIdentity:
  myid: abc
  mysecondid: efg
labels:
  mylabel: efg
  mysecondlabel:
    key: true
EOF > ./comp-ref.yaml
$ component-cli component-reference add -r ./comp-ref.yaml
```

```shell script
# define by commandline
$ component-cli component-reference add \
  --name="cli" \
  --componentName="github.com/gardener/component-spec" \
  --version="v0.0.1" \
  --identity="myid=abc" \
  --identity="mysecondid=efg" \
  --label="mylabel=efg" \
  --label="mysecondlabel={\"key\": true}"
```

## Remote

The `remote` subcommand contains utility functions to interact with component referneces stored in a remote component respoitory (oci repository).

:warning: Currently the component-cli uses the default docker config for authentication. 
Use `docker login` to authenticate against a oci repository.

### Push

A component descriptor in component archive CTF format can be pushed to an component repository(oci repository) by using the `remote push` command.

This command takes 1 argument which is the path to the component archive.<br>
A component archive is a directory that contains the component descriptor at `/component-descriptor.yaml` and all blobs at `/blobs/<blobname>`.

```shell script
$ component-cli remote push [path to component archive]
```