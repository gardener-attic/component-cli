// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0
package signatures

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
)

// CreateAndVerifyX509CertificateFromFiles creates and verifies a x509 certificate from certificate files.
// The certificates must be in PEM format.
func CreateAndVerifyX509CertificateFromFiles(certPath, intermediateCAsCertsPath, rootCACertPath string) (*x509.Certificate, error) {
	var err error

	var rootCACert []byte
	if rootCACertPath != "" {
		rootCACert, err = ioutil.ReadFile(rootCACertPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read root CA certificate file: %w", err)
		}
	}

	var intermediateCAsCerts []byte
	if intermediateCAsCertsPath != "" {
		intermediateCAsCerts, err = ioutil.ReadFile(intermediateCAsCertsPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read intermediate CAs certificates file: %w", err)
		}
	}

	cert, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read certificate file: %w", err)
	}

	return CreateAndVerifyX509Certificate(cert, intermediateCAsCerts, rootCACert)
}

// CreateAndVerifyX509Certificate creates and verifies a x509 certificate from in-memory raw certificates.
// The certificates must be in PEM format.
func CreateAndVerifyX509Certificate(cert, intermediateCAsCerts, rootCACert []byte) (*x509.Certificate, error) {
	// First, create the set of root certificates. For this example we only
	// have one. It's also possible to omit this in order to use the
	// default root set of the current operating system.
	var roots *x509.CertPool
	if rootCACert != nil {
		roots = x509.NewCertPool()
		if ok := roots.AppendCertsFromPEM(rootCACert); !ok {
			return nil, errors.New("unable to parse root ca certificate")
		}
	}

	var intermediates *x509.CertPool
	if intermediateCAsCerts != nil {
		intermediates = x509.NewCertPool()
		if ok := intermediates.AppendCertsFromPEM(intermediateCAsCerts); !ok {
			return nil, errors.New("unable to parse intermediate cas certificates")
		}
	}

	block, _ := pem.Decode(cert)
	if block == nil {
		return nil, errors.New("unable to decode certificate")
	}
	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate: %w", err)
	}

	opts := x509.VerifyOptions{
		// TODO: must we set DNSName option?
		// DNSName:       "mail.google.com",
		Roots:         roots,
		Intermediates: intermediates,
	}

	if _, err := parsedCert.Verify(opts); err != nil {
		return nil, fmt.Errorf("unable to verify certificate: %w", err)
	}

	return parsedCert, nil
}
