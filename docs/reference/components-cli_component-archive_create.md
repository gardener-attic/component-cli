## components-cli component-archive create

Creates a component archive with a component descriptor

### Synopsis


Create command creates a new component archive directory with a "component-descriptor.yaml" file.


```
components-cli component-archive create [component-archive-path] [flags]
```

### Options

```
      --format CAOutputFormat   output format of the component archive. Can be "fs", "tar" or "tgz"
  -h, --help                    help for create
  -o, --out string              writes the resulting archive to the given path
```

### Options inherited from parent commands

```
      --cli                  logger runs as cli logger. enables cli logging
      --dev                  enable development logging which result in console encoding, enabled stacktrace and enabled caller
      --disable-caller       disable the caller of logs (default true)
      --disable-stacktrace   disable the stacktrace of error logs (default true)
      --disable-timestamp    disable timestamp output (default true)
  -v, --verbosity int        number for the log level verbosity (default 1)
```

### SEE ALSO

* [components-cli component-archive](components-cli_component-archive.md)	 - 

