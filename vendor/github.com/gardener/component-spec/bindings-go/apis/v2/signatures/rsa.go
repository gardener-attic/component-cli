package signatures

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"

	v2 "github.com/gardener/component-spec/bindings-go/apis/v2"
)

// RsaSigner is a signatures.Signer compatible struct to sign with RSASSA-PKCS1-V1_5-SIGN.
type RsaSigner struct {
	privateKey rsa.PrivateKey
}

// CreateRsaSignerFromKeyFile creates an Instance of RsaSigner with the given private key.
// The private key has to be in the PKCS #1, ASN.1 DER form, see x509.ParsePKCS1PrivateKey.
func CreateRsaSignerFromKeyFile(pathToPrivateKey string) (*RsaSigner, error) {
	privKeyFile, err := ioutil.ReadFile(pathToPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed opening private key file %w", err)
	}

	block, _ := pem.Decode([]byte(privKeyFile))
	if block == nil {
		return nil, fmt.Errorf("failed decoding PEM formatted block in key %w", err)
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed parsing key %w", err)
	}
	return &RsaSigner{
		privateKey: *key,
	}, nil
}

// Sign returns the signature for the data for the component-descriptor.
func (s RsaSigner) Sign(componentDescriptor v2.ComponentDescriptor, digest v2.DigestSpec) (*v2.SignatureSpec, error) {
	decodedHash, err := hex.DecodeString(digest.Value)
	if err != nil {
		return nil, fmt.Errorf("failed decoding hash to bytes")
	}
	hashType, err := hashAlgorithmLookup(digest.HashAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("failed looking up hash algorithm")
	}
	signature, err := rsa.SignPKCS1v15(nil, &s.privateKey, hashType, decodedHash)
	if err != nil {
		return nil, fmt.Errorf("failed signing hash, %w", err)
	}
	return &v2.SignatureSpec{
		Algorithm: "RSASSA-PKCS1-V1_5-SIGN",
		Value:     hex.EncodeToString(signature),
	}, nil
}

// maps a hashing algorithm string to crypto.Hash
func hashAlgorithmLookup(algorithm string) (crypto.Hash, error) {
	switch strings.ToLower(algorithm) {
	case SHA256:
		return crypto.SHA256, nil
	}
	return 0, fmt.Errorf("hash Algorithm %s not found", algorithm)
}

// RsaVerifier is a signatures.Verifier compatible struct to verify RSASSA-PKCS1-V1_5-SIGN signatures.
type RsaVerifier struct {
	publicKey rsa.PublicKey
}

// CreateRsaVerifierFromKeyFile creates an Instance of RsaVerifier with the given rsa public key.
// The private key has to be in the PKIX, ASN.1 DER form, see x509.ParsePKIXPublicKey.
func CreateRsaVerifierFromKeyFile(pathToPublicKey string) (*RsaVerifier, error) {
	publicKey, err := ioutil.ReadFile(pathToPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed opening public key file %w", err)
	}
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return nil, fmt.Errorf("failed decoding PEM formatted block in key %w", err)
	}
	untypedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed parsing key %w", err)
	}
	switch key := untypedKey.(type) {
	case *rsa.PublicKey:
		return &RsaVerifier{
			publicKey: *key,
		}, nil
	default:
		return nil, fmt.Errorf("public key format is not supported. Only rsa.PublicKey is supported")
	}
}

// Verify checks the signature, returns an error on verification failure
func (v RsaVerifier) Verify(componentDescriptor v2.ComponentDescriptor, signature v2.Signature) error {
	decodedHash, err := hex.DecodeString(signature.Digest.Value)
	if err != nil {
		return fmt.Errorf("failed decoding hash %s: %w", signature.Digest.Value, err)
	}
	decodedSignature, err := hex.DecodeString(signature.Signature.Value)
	if err != nil {
		return fmt.Errorf("failed decoding hash %s: %w", signature.Digest.Value, err)
	}
	algorithm, err := hashAlgorithmLookup(signature.Digest.HashAlgorithm)
	if err != nil {
		return fmt.Errorf("failed looking up hash algorithm for %s: %w", signature.Digest.HashAlgorithm, err)
	}
	err = rsa.VerifyPKCS1v15(&v.publicKey, algorithm, decodedHash, decodedSignature)
	if err != nil {
		return fmt.Errorf("signature verification failed, %w", err)
	}
	return nil
}
