// Package certpin converts "ignore certificate verification" into the
// certificate pinning required by current xray-core releases.
package certpin

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/krabt/krab/pkg/profile"
)

// Resolve fetches the server's leaf certificate without validating its trust
// chain, then pins that exact certificate for the subsequent xray connection.
func Resolve(server profile.Server) (profile.Server, error) {
	if server.Extra["insecure"] != "1" && server.Extra["insecure"] != "true" {
		return server, nil
	}
	// The insecure flag can remain in imported or previously edited profiles
	// after their security mode changes. Plain and REALITY connections don't
	// perform a conventional TLS handshake and therefore have no certificate
	// to read or pin here.
	security := server.Extra["security"]
	if security == "" {
		security = server.Extra["tls"]
	}
	if server.Protocol != "hysteria2" && security != "tls" {
		return server, nil
	}
	extra := make(map[string]string, len(server.Extra)+1)
	for key, value := range server.Extra {
		extra[key] = value
	}
	delete(extra, "insecure")
	if extra["pinSHA256"] != "" {
		server.Extra = extra
		return server, nil
	}

	serverName := extra["sni"]
	if serverName == "" {
		serverName = server.Address
	}
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(server.Address, strconv.Itoa(server.Port)), &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true, // Required only to read a self-signed certificate for pinning.
	})
	if err != nil {
		return server, fmt.Errorf("read server certificate for pinning: %w", err)
	}
	defer conn.Close()
	certificates := conn.ConnectionState().PeerCertificates
	if len(certificates) == 0 {
		return server, fmt.Errorf("read server certificate for pinning: server returned no certificate")
	}
	hash := sha256.Sum256(certificates[0].Raw)
	extra["pinSHA256"] = hex.EncodeToString(hash[:])
	server.Extra = extra
	return server, nil
}
