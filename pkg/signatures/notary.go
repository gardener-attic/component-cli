package signatures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/ghodss/yaml"
)

type NotarySigner struct {
	Url string
	Jwt string
}
type NoatrySignPayload struct {
	NotaryGun string `json:"notaryGun"`
	Sha256    string `json:"sha256"`
	ByteSize  int    `json:"byteSize"`
	Version   string `json:"version"`
}

func CreateNotarySignerFromConfig(configPath string) (*NotarySigner, error) {
	configContent, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed opening config file %w", err)
	}

	var config struct {
		Url string `json:"url"`
		Jwt string `json:"jwt"`
	}
	err = yaml.Unmarshal(configContent, &config)
	if err != nil {
		return nil, fmt.Errorf("failed parsing config file %w", err)
	}
	return &NotarySigner{
		Url: config.Url,
		Jwt: config.Jwt,
	}, nil
}

// Sign returns the signature for the data for the component-descriptor.
func (s NotarySigner) Sign(componentDescriptor v2.ComponentDescriptor, digest v2.DigestSpec) (*v2.SignatureSpec, error) {
	if strings.ToUpper(digest.HashAlgorithm) != "SHA256" {
		return nil, fmt.Errorf("hash algorithm %s not supported. Only SHA256 allowed", digest.HashAlgorithm)
	}

	notarySignPayload := []NoatrySignPayload{
		{
			NotaryGun: fmt.Sprintf("%s/gardener-poc/%s/%s", "common.repositories.cloud.sap", componentDescriptor.Name, componentDescriptor.Version),
			Sha256:    digest.Value,
			ByteSize:  len(digest.Value), //TODO: use a correct byte size
			Version:   componentDescriptor.Version,
		},
	}
	marshaledNotarySignPayload, err := json.Marshal(notarySignPayload)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling payload to json: %w", err)
	}

	req, err := http.NewRequest("POST", s.Url, bytes.NewReader(marshaledNotarySignPayload))
	if err != nil {
		return nil, fmt.Errorf("failed calling notary to sign %s:%s: %w", componentDescriptor.Name, componentDescriptor.Version, err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.Jwt))
	req.Header.Set("Content-type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed doing request: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %w", err)
	}
	fmt.Print(string(body))

	return &v2.SignatureSpec{
		Algorithm: "Notary",
		Value:     "",
	}, nil
}

type NotaryVerifier struct {
	Url     string
	DssPath string
}

func CreateNotaryVerifierFromConfig(configPath string) (*NotaryVerifier, error) {
	configContent, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed opening config file %w", err)
	}

	var config struct {
		Url     string `json:"verifyUrl"`
		DssPath string `json:"dss"`
	}
	err = yaml.Unmarshal(configContent, &config)
	if err != nil {
		return nil, fmt.Errorf("failed parsing config file %w", err)
	}
	return &NotaryVerifier{
		Url:     config.Url,
		DssPath: config.DssPath,
	}, nil
}

func (v NotaryVerifier) Verify(componentDescriptor v2.ComponentDescriptor, signature v2.Signature) error {
	// if signature.Signature.Algorithm != "NOTARY" {
	// 	return fmt.Errorf("Signature algorithm %s not supported by NotaryVerifier", signature.Signature.Algorithm)
	// }
	notaryGun := fmt.Sprintf("%s/gardener-poc/%s/%s", "common.repositories.cloud.sap", componentDescriptor.Name, componentDescriptor.Version)
	notaryVerifyCmd := exec.Command(v.DssPath, "lookup", "-s", v.Url, notaryGun, componentDescriptor.Version)
	output, err := notaryVerifyCmd.Output()
	if err != nil {
		return fmt.Errorf("failed calling notary dss executable: %w", err)
	}
	outputString := strings.Trim(string(output), " \n")
	splited := strings.Split(outputString, " ")
	if len(splited) != 3 {
		return fmt.Errorf("response from notary dss executable could not be read: %s", outputString)
	}
	version := splited[0]
	hashWithAlgorithm := splited[1]
	// length := splited[2]

	if version != componentDescriptor.Version {
		return fmt.Errorf("version in notary response missmatches cd version")
	}

	hashSplited := strings.Split(hashWithAlgorithm, ":")
	if len(hashSplited) != 2 {
		return fmt.Errorf("hashWithAlgorithm could not be separated into algorithm and hash digest: %s", hashWithAlgorithm)
	}
	hashAlgorithm := hashSplited[0]
	hashValue := hashSplited[1]

	if !strings.EqualFold(hashAlgorithm, signature.Digest.HashAlgorithm) {
		return fmt.Errorf("hash algorithm from notary %s missmatches signature hash algorithm %s", hashAlgorithm, signature.Digest.HashAlgorithm)
	}
	if !strings.EqualFold(hashValue, signature.Digest.Value) {
		return fmt.Errorf("hash digest from notary %s missmatches signature hash digest %s", hashValue, signature.Digest.Value)
	}

	return nil
}
