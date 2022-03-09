package signatures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"sigs.k8s.io/yaml"
)

type SignServerSigner struct {
	Url      string
	Username string
	Password string
}

func NewSignServerFromConfigFile(configFilePath string) (*SignServerSigner, error) {
	configFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed reading config file: %w", err)
	}
	var signServer SignServerSigner
	if err := yaml.Unmarshal(configFile, &signServer); err != nil {
		return nil, fmt.Errorf("failed parsing config yaml: %w", err)
	}
	return &signServer, nil
}

func (signer *SignServerSigner) Sign(componentDescriptor v2.ComponentDescriptor, digest v2.DigestSpec) (*v2.SignatureSpec, error) {
	postBody := struct {
		Digest string `json:"digest"`
	}{
		Digest: fmt.Sprintf("%s:%s", strings.ToLower(digest.HashAlgorithm), digest.Value),
	}
	jsonPostBody, err := json.Marshal(postBody)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling http request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/sign-digest", signer.Url), bytes.NewBuffer(jsonPostBody))
	if err != nil {
		return nil, fmt.Errorf("failed building http request: %w", err)
	}
	req.SetBasicAuth(signer.Username, signer.Password)

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed sending request to signing service: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		if err != nil {
			return nil, fmt.Errorf("failed sending request to signing service: received status code %d, expected 200", res.StatusCode)
		}
	}

	resBodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading signing service request response body: %w", err)
	}

	var responseBody struct {
		Digest      string `json:"digest"`
		Signature   string `json:"signature"`
		Certificate string `json:"certificate"`
	}
	if err := json.Unmarshal(resBodyBytes, &responseBody); err != nil {
		return nil, fmt.Errorf("failed unmarshaling signing service request response body: %w", err)
	}

	if responseBody.Digest != fmt.Sprintf("%s:%s", strings.ToLower(digest.HashAlgorithm), digest.Value) || responseBody.Signature == "" || responseBody.Certificate == "" {
		return nil, fmt.Errorf("signing service request response incomplete: %+v", responseBody)
	}

	return &v2.SignatureSpec{
		Algorithm: "SIGN-SERVER-RSASSA-PKCS1-V1_5-SIGN/V1",
		Value:     responseBody.Signature,
	}, nil

}
