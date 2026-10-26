package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
)

// A server that checks client certificates refuses one marked only for server
// auth, which is all sing-box's own generator makes. The pair also has to load
// as one: the frontend splits the lines at the PEM markers, key first.
func TestTLSClientKeypairIsForClientAuth(t *testing.T) {
	lines, err := (&ServerService{}).GenKeypair("tls-client", "vpn-user")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(lines, "\n")
	if !strings.HasPrefix(joined, "-----BEGIN PRIVATE KEY-----") {
		t.Fatalf("want the key first, got %q", lines[0])
	}

	keyBlock, rest := pem.Decode([]byte(joined))
	certBlock, _ := pem.Decode(rest)
	if keyBlock == nil || certBlock == nil || certBlock.Type != "CERTIFICATE" {
		t.Fatalf("want a key then a certificate, got %q", joined)
	}
	if _, err := tls.X509KeyPair(pem.EncodeToMemory(certBlock), pem.EncodeToMemory(keyBlock)); err != nil {
		t.Fatalf("key and certificate do not pair: %v", err)
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(cert.ExtKeyUsage) != 1 || cert.ExtKeyUsage[0] != x509.ExtKeyUsageClientAuth {
		t.Errorf("ext key usage = %v, want client auth only", cert.ExtKeyUsage)
	}
	if cert.Subject.CommonName != "vpn-user" {
		t.Errorf("common name = %q, want vpn-user", cert.Subject.CommonName)
	}
	if len(cert.DNSNames) != 0 {
		t.Errorf("a client certificate names no host, got SANs %v", cert.DNSNames)
	}
}

// A private key that does not parse has no public half. Answering with one
// anyway -- the zero key's, or an error message as the first line -- lands in
// the endpoint drawer's public_key field and is saved as the key.
func TestWireGuardPublicKeyOfBadPrivateKeyFails(t *testing.T) {
	lines, err := (&ServerService{}).GenKeypair("wireguard", "not-a-key")
	if err == nil {
		t.Fatalf("want an error, got %q", lines)
	}
	if lines != nil {
		t.Errorf("want no lines with the error, got %q", lines)
	}
}

// A type with no generator is an error too, not a line of text the drawer would
// take for a key.
func TestKeypairOfUnknownTypeFails(t *testing.T) {
	for _, keyType := range []string{"", "rsa"} {
		lines, err := (&ServerService{}).GenKeypair(keyType, "")
		if err == nil || lines != nil {
			t.Errorf("type %q: want an error and no lines, got %q, %v", keyType, lines, err)
		}
	}
}

// The drawers used to send the literal string '' for an empty server name.
// An ECH config whose public name is not a domain is skipped by Go's client
// (crypto/tls pickECHConfig), and a self-signed certificate for "" is valid for
// no name, so both are refused instead of generated.
func TestKeypairsNeedAUsableServerName(t *testing.T) {
	server := &ServerService{}
	for _, name := range []string{"", "''", "myserver", "-bad.example.com", "a..b"} {
		if lines, err := server.GenKeypair("ech", name); err == nil {
			t.Errorf("ech %q: want an error, got %d lines", name, len(lines))
		}
	}
	for _, name := range []string{"", "  "} {
		if lines, err := server.GenKeypair("tls", name); err == nil {
			t.Errorf("tls %q: want an error, got %d lines", name, len(lines))
		}
	}
	if _, err := server.GenKeypair("ech", "example.com"); err != nil {
		t.Errorf("ech example.com: %v", err)
	}
}

// A pasted name with a stray space is used without it: the certificate is for
// the domain, not for " example.com", and ECH accepts it rather than refusing.
func TestKeypairServerNameIsTrimmed(t *testing.T) {
	server := &ServerService{}
	lines, err := server.GenKeypair("tls", " example.com ")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(lines, "\n")
	_, rest := pem.Decode([]byte(joined))
	block, _ := pem.Decode(rest)
	if block == nil {
		t.Fatalf("no certificate in %q", joined)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if len(cert.DNSNames) != 1 || cert.DNSNames[0] != "example.com" {
		t.Errorf("SANs = %q, want [example.com]", cert.DNSNames)
	}
	if _, err := server.GenKeypair("ech", " example.com "); err != nil {
		t.Errorf("ech with a stray space: %v", err)
	}
}

func TestWireGuardPublicKeyMatchesGeneratedPair(t *testing.T) {
	server := &ServerService{}
	pair, err := server.GenKeypair("wireguard", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(pair) != 2 || !strings.HasPrefix(pair[0], "PrivateKey: ") || !strings.HasPrefix(pair[1], "PublicKey: ") {
		t.Fatalf("unexpected pair %q", pair)
	}
	public, err := server.GenKeypair("wireguard", strings.TrimPrefix(pair[0], "PrivateKey: "))
	if err != nil {
		t.Fatal(err)
	}
	if len(public) != 1 || public[0] != strings.TrimPrefix(pair[1], "PublicKey: ") {
		t.Errorf("derived %q, want %q", public, strings.TrimPrefix(pair[1], "PublicKey: "))
	}
}
