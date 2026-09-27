package certpin

import (
	"reflect"
	"testing"

	"github.com/krabt/krab/pkg/profile"
)

func TestResolveSkipsCertificatePinningWithoutTLS(t *testing.T) {
	tests := []struct {
		name     string
		security string
	}{
		{name: "none", security: "none"},
		{name: "reality", security: "reality"},
		{name: "unset"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := profile.Server{
				Protocol: "vless",
				Address:  "127.0.0.1",
				Port:     1,
				Extra: map[string]string{
					"security": test.security,
					"insecure": "true",
				},
			}
			got, err := Resolve(server)
			if err != nil {
				t.Fatalf("Resolve() unexpectedly tried a TLS handshake: %v", err)
			}
			if !reflect.DeepEqual(got, server) {
				t.Fatalf("Resolve() changed a non-TLS server: got %#v, want %#v", got, server)
			}
		})
	}
}
