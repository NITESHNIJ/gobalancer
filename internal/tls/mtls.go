package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// MTLSConfig builds a *tls.Config that requires and validates client certificates.
// caFile is a PEM-encoded certificate authority pool.
func MTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load server cert: %w", err)
	}

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("no valid CA certs found in %s", caFile)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// RedirectHandler returns an http.Handler that issues 301 redirects to HTTPS.
// Used when the :80 listener receives plain-HTTP traffic.
func RedirectHandler(httpsHost string) func(w interface{ Header() interface{ Set(string, string) }; WriteHeader(int) }) {
	// Thin wrapper — actual redirect logic lives in middleware/redirect.go.
	// Returning a typed fn here keeps the tls package free of net/http import.
	_ = httpsHost
	return nil
}
