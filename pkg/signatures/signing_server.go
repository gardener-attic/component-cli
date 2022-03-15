// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package signatures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/yaml"
)

type SigningServerSigner struct {
	Url      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewSigningServerSignerFromConfigFile(configFilePath string) (*SigningServerSigner, error) {
	configBytes, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed reading config file: %w", err)
	}
	var signer SigningServerSigner
	if err := yaml.Unmarshal(configBytes, &signer); err != nil {
		return nil, fmt.Errorf("failed parsing config yaml: %w", err)
	}
	return &signer, nil
}

func (signer *SigningServerSigner) Sign(componentDescriptor cdv2.ComponentDescriptor, digest cdv2.DigestSpec) (*cdv2.SignatureSpec, error) {
	requestBody := struct {
		Digest string `json:"digest"`
	}{
		Digest: fmt.Sprintf("%s:%s", strings.ToLower(digest.HashAlgorithm), digest.Value),
	}

	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/sign-digest", signer.Url), bytes.NewBuffer(requestBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed building http request: %w", err)
	}
	req.SetBasicAuth(signer.Username, signer.Password)

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed sending request: %w", err)
	}
	defer res.Body.Close()

	responseBodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request returned with response code %d: %s", res.StatusCode, string(responseBodyBytes))
	}

	var responseBody struct {
		Digest    string `json:"digest"`
		Signature string `json:"signature"`
	}
	if err := json.Unmarshal(responseBodyBytes, &responseBody); err != nil {
		return nil, fmt.Errorf("failed unmarshaling response body: %w", err)
	}

	if responseBody.Digest != fmt.Sprintf("%s:%s", strings.ToLower(digest.HashAlgorithm), digest.Value) || responseBody.Signature == "" {
		return nil, fmt.Errorf("invalid signing server response: %+v", responseBody)
	}

	return &cdv2.SignatureSpec{
		Algorithm: "SIGN-SERVER-RSASSA-PKCS1-V1_5-SIGN/V1",
		Value:     responseBody.Signature,
	}, nil
}
