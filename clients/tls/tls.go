// Copyright 2026 Sven Victor
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package tls builds crypto/tls.Config values from TLSOptions, including CA
// certificates, client certificates, and a minimum protocol version.
package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Versions maps TLS version names used in TLSOptions.MinVersion to protocol
// constants. Recognized keys are TLS13, TLS12, TLS11, and TLS10.
var Versions = map[string]uint16{
	"TLS13": tls.VersionTLS13,
	"TLS12": tls.VersionTLS12,
	"TLS11": tls.VersionTLS11,
	"TLS10": tls.VersionTLS10,
}

func readCAFile(f string) ([]byte, error) {
	data, err := os.ReadFile(f)
	if err != nil {
		return nil, fmt.Errorf("unable to load specified CA cert %s: %s", f, err)
	}
	return data, nil
}

// updateRootCA parses the given byte slice as a series of PEM encoded certificates and updates tls.Config.RootCAs.
func updateRootCA(cfg *tls.Config, b []byte) bool {
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(b) {
		return false
	}
	cfg.RootCAs = caCertPool
	return true
}

type TLSOptions struct {
	CAFile             string `json:"ca_file,omitempty" yaml:"ca_file,omitempty" mapstructure:"ca_file"`
	CertFile           string `json:"cert_file,omitempty" yaml:"cert_file,omitempty" mapstructure:"cert_file"`
	KeyFile            string `json:"key_file,omitempty" yaml:"key_file,omitempty" mapstructure:"key_file"`
	ServerName         string `json:"server_name,omitempty" yaml:"server_name,omitempty" mapstructure:"server_name"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty" yaml:"insecure_skip_verify,omitempty" mapstructure:"insecure_skip_verify"`
	MinVersion         string `json:"min_version,omitempty" yaml:"min_version,omitempty" mapstructure:"min_version"`
}

// getClientCertificate reads the pair of client cert and key from disk and returns a tls.Certificate.
func (c *TLSOptions) getClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("unable to use specified client cert (%s) & key (%s): %s", c.CertFile, c.KeyFile, err)
	}
	return &cert, nil
}

// NewTLSConfig creates a crypto/tls.Config from TLSOptions.
// A nil cfg yields a default tls.Config. CAFile, CertFile, and KeyFile are
// read from disk; MinVersion must be a key in Versions (for example "TLS12").
func NewTLSConfig(cfg *TLSOptions) (*tls.Config, error) {
	if cfg == nil {
		return &tls.Config{}, nil
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		MinVersion:         Versions[cfg.MinVersion],
	}

	// If a CA cert is provided then let's read it in so we can validate the
	// scrape target's certificate properly.
	if len(cfg.CAFile) > 0 {
		b, err := readCAFile(cfg.CAFile)
		if err != nil {
			return nil, err
		}
		if !updateRootCA(tlsConfig, b) {
			return nil, fmt.Errorf("unable to use specified CA cert %s", cfg.CAFile)
		}
	}

	if len(cfg.ServerName) > 0 {
		tlsConfig.ServerName = cfg.ServerName
	}
	// If a client cert & key is provided then configure TLS config accordingly.
	if len(cfg.CertFile) > 0 && len(cfg.KeyFile) == 0 {
		return nil, fmt.Errorf("client cert file %q specified without client key file", cfg.CertFile)
	} else if len(cfg.KeyFile) > 0 && len(cfg.CertFile) == 0 {
		return nil, fmt.Errorf("client key file %q specified without client cert file", cfg.KeyFile)
	} else if len(cfg.CertFile) > 0 && len(cfg.KeyFile) > 0 {
		// Verify that client cert and key are valid.
		if _, err := cfg.getClientCertificate(nil); err != nil {
			return nil, err
		}
		tlsConfig.GetClientCertificate = cfg.getClientCertificate
	}

	return tlsConfig, nil
}
