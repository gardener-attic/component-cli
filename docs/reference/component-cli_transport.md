## component-cli transport



```
component-cli transport [flags]
```

### Options

```
      --allow-plain-http               allows the fallback to http if the oci registry does not support https
      --cc-config string               path to the local concourse config file
      --dry-run                        only download component descriptors and perform matching of resources against transport config file. no component descriptors are uploaded, no resources are down/uploaded
      --from string                    source repository base url
  -h, --help                           help for transport
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --processor-timeout duration     execution timeout for each individual processor (default 30s)
      --registry-config string         path to the dockerconfig.json with the oci registry authentication information
      --repo-ctx-override-cfg string   path to the repository context override config file
      --to string                      target repository where the components are copied to
      --transport-cfg string           path to the transport config file
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

* [component-cli](component-cli.md)	 - component cli

