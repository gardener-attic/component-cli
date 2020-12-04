// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package credentials

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	dockerconfig "github.com/docker/cli/cli/config"
	dockercreds "github.com/docker/cli/cli/config/credentials"
	dockerconfigtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/distribution/reference"
	corev1 "k8s.io/api/core/v1"
)

// OCIKeyring is the interface that implements are keyring to retrieve credentials for a given
// server.
type OCIKeyring interface {
	// Get retrieves credentials from the keyring for a given resource url.
	Get(resourceURl string) (dockerconfigtypes.AuthConfig, bool)
	// Resolver returns a new authenticated resolver.
	Resolver(ctx context.Context, ref string, client *http.Client, plainHTTP bool) (remotes.Resolver, error)
}

// CreateOCIRegistryKeyring creates a new OCI registry keyring.
func CreateOCIRegistryKeyring(pullSecrets []corev1.Secret, configFiles []string) (*GeneralOciKeyring, error) {
	store := New()
	for _, secret := range pullSecrets {
		if secret.Type != corev1.SecretTypeDockerConfigJson {
			continue
		}
		dockerConfigBytes, ok := secret.Data[corev1.DockerConfigJsonKey]
		if !ok {
			continue
		}

		dockerConfig, err := dockerconfig.LoadFromReader(bytes.NewBuffer(dockerConfigBytes))
		if err != nil {
			return nil, err
		}

		// currently only support the default credential store.
		credStore := dockerConfig.GetCredentialsStore("")
		if err := store.Add(credStore); err != nil {
			return nil, err
		}
	}

	for _, configFile := range configFiles {
		dockerConfigBytes, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}

		dockerConfig, err := dockerconfig.LoadFromReader(bytes.NewBuffer(dockerConfigBytes))
		if err != nil {
			return nil, err
		}

		// currently only support the default credential store.
		credStore := dockerConfig.GetCredentialsStore("")
		if err := store.Add(credStore); err != nil {
			return nil, err
		}
	}

	return store, nil
}

// GeneralOciKeyring is general implementation of a oci keyring that can be extended with other credentials.
type GeneralOciKeyring struct {
	// index is an additional index structure that also contains multi
	index *IndexNode
	store map[string]dockerconfigtypes.AuthConfig
}

type IndexNode struct {
	Segment  string
	Address  string
	Children []*IndexNode
}

func (n *IndexNode) Set(path, address string) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 0 || (len(splitPath) == 1 && len(splitPath[0]) == 0) {
		n.Address = address
		return
	}
	child := n.FindSegment(splitPath[0])
	if child == nil {
		child = &IndexNode{
			Segment: splitPath[0],
		}
		n.Children = append(n.Children, child)
	}
	child.Set(strings.Join(splitPath[1:], "/"), address)
}

func (n *IndexNode) FindSegment(segment string) *IndexNode {
	for _, child := range n.Children {
		if child.Segment == segment {
			return child
		}
	}
	return nil
}

func (n *IndexNode) Find(path string) (string, bool) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 0 || (len(splitPath) == 1 && len(splitPath[0]) == 0) {
		return n.Address, true
	}
	child := n.FindSegment(splitPath[0])
	if child == nil {
		// returns the current address if no more specific auth config is defined
		return n.Address, true
	}
	return child.Find(strings.Join(splitPath[1:], "/"))
}

// New creates a new empty general oci keyring.
func New() *GeneralOciKeyring {
	return &GeneralOciKeyring{
		index: &IndexNode{},
		store: make(map[string]dockerconfigtypes.AuthConfig),
	}
}

var _ OCIKeyring = &GeneralOciKeyring{}

// Size returns the size of the keyring
func (o GeneralOciKeyring) Size() int {
	return len(o.store)
}

func (o GeneralOciKeyring) Get(resourceURl string) (dockerconfigtypes.AuthConfig, bool) {
	ref, err := reference.ParseNamed(resourceURl)
	if err == nil {
		// if the name is not conical try to treat it like a host name
		resourceURl = ref.Name()
	}
	address, ok := o.index.Find(resourceURl)
	if !ok {
		return dockerconfigtypes.AuthConfig{}, false
	}
	if auth, ok := o.store[address]; ok {
		return auth, ok
	}
	return dockerconfigtypes.AuthConfig{}, false
}

// getCredentials returns the username and password for a given url.
// It implements the Credentials func for a docker resolver
func (o *GeneralOciKeyring) getCredentials(url string) (string, string, error) {
	auth, ok := o.Get(url)
	if !ok {
		return "", "", fmt.Errorf("authentication for %s cannot be found", url)
	}

	return auth.Username, auth.Password, nil
}

// AddAuthConfig adds a auth config for a address
func (o *GeneralOciKeyring) AddAuthConfig(address string, auth dockerconfigtypes.AuthConfig) error {
	// normalize host name
	var err error
	address, err = normalizeHost(address)
	if err != nil {
		return err
	}
	o.store[address] = auth
	o.index.Set(address, address)
	return nil
}

// Add adds all addresses of a docker credential store.
func (o *GeneralOciKeyring) Add(store dockercreds.Store) error {
	auths, err := store.GetAll()
	if err != nil {
		return err
	}
	for address, auth := range auths {
		if err := o.AddAuthConfig(address, auth); err != nil {
			return err
		}
	}
	return nil
}

func (o *GeneralOciKeyring) Resolver(ctx context.Context, ref string, client *http.Client, plainHTTP bool) (remotes.Resolver, error) {
	if ref == "" {
		return docker.NewResolver(docker.ResolverOptions{
			Credentials: o.getCredentials,
			Client:      client,
			PlainHTTP:   plainHTTP,
		}), nil
	}

	// get specific auth for ref and only return a resolver with that authentication config
	auth, ok := o.Get(ref)
	if !ok {
		return docker.NewResolver(docker.ResolverOptions{
			Credentials: o.getCredentials,
			Client:      client,
			PlainHTTP:   plainHTTP,
		}), nil
	}
	return docker.NewResolver(docker.ResolverOptions{
		Credentials: func(_ string) (string, string, error) {
			return auth.Username, auth.Password, nil
		},
		Client:    client,
		PlainHTTP: plainHTTP,
	}), nil
}

func normalizeHost(u string) (string, error) {
	if !strings.Contains(u, "://") {
		u = "dummy://" + u
	}
	host, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return path.Join(host.Host, host.Path), nil
}
