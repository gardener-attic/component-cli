module github.com/gardener/component-cli

go 1.15

require (
	github.com/containerd/containerd v1.4.2
	github.com/deislabs/oras v0.8.1
	github.com/docker/cli v20.10.0-rc1+incompatible
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/gardener/component-spec/bindings-go v0.0.0-20201127224544-4fd6d604604f
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.3.0
	github.com/golang/mock v1.4.4
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/mandelsoft/vfs v0.0.0-20201002134249-3c471f64a4d1
	github.com/onsi/ginkgo v1.14.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/prometheus/client_golang v1.5.1 // indirect
	github.com/prometheus/procfs v0.0.11 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b
	golang.org/x/tools v0.0.0-20201002184944-ecd9fd270d5d // indirect
	gotest.tools/v3 v3.0.3 // indirect
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/yaml v1.2.0
)
