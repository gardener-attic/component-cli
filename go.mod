module github.com/gardener/component-cli

go 1.15

require (
	github.com/containerd/containerd v1.4.2
	github.com/docker/cli v20.10.0-rc1+incompatible
	github.com/docker/docker v1.4.2-0.20200203170920-46ec8731fbce // indirect
	github.com/drone/envsubst v1.0.2
	github.com/gardener/component-spec/bindings-go v0.0.53
	github.com/gardener/image-vector v0.5.0
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.3.0
	github.com/golang/mock v1.5.0
	github.com/google/go-containerregistry v0.5.0
	github.com/google/uuid v1.1.2
	github.com/mandelsoft/vfs v0.0.0-20210530103237-5249dc39ce91
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.13.0
	github.com/opencontainers/distribution-spec v1.0.0-rc1
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/prometheus/client_golang v0.9.3
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.2.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	gotest.tools/v3 v3.0.3 // indirect
	k8s.io/api v0.19.4
	k8s.io/apimachinery v0.19.4
	sigs.k8s.io/yaml v1.2.0
)
