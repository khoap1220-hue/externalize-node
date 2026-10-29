package hosted

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFromEnvDefaults(t *testing.T) {
	t.Setenv("EXNODE_RPC", "https://rpc.example")
	h, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "https://externalize-labs.github.io")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var body struct {
		Network string `json:"network"`
		RPC     bool   `json:"rpc"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("status %d, body %q", rec.Code, rec.Body)
	}
	if body.Network != "Public Global Stellar Network ; September 2015" || !body.RPC {
		t.Fatalf("health %+v", body)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("a hosted node answers browsers on any origin by default")
	}
}

func TestFromEnvRejectsUnknownNetwork(t *testing.T) {
	t.Setenv("EXNODE_NETWORK", "moonnet")
	if _, err := FromEnv(); err == nil {
		t.Fatal("unknown network accepted")
	}
}

func TestFromEnvConfiguration(t *testing.T) {
	cases := []struct {
		name        string
		env         map[string]string
		origin      string
		wantNetwork string
		wantRPC     bool
		wantCORS    string
	}{
		{
			name:        "cors allows first listed origin",
			env:         map[string]string{"EXNODE_CORS": "https://a.example, https://b.example", "EXNODE_RPC": "https://rpc.example"},
			origin:      "https://a.example",
			wantNetwork: "Public Global Stellar Network ; September 2015",
			wantRPC:     true,
			wantCORS:    "https://a.example",
		},
		{
			name:        "cors allows second listed origin",
			env:         map[string]string{"EXNODE_CORS": "https://a.example, https://b.example", "EXNODE_RPC": "https://rpc.example"},
			origin:      "https://b.example",
			wantNetwork: "Public Global Stellar Network ; September 2015",
			wantRPC:     true,
			wantCORS:    "https://b.example",
		},
		{
			name:        "cors omits unlisted origin",
			env:         map[string]string{"EXNODE_CORS": "https://a.example, https://b.example", "EXNODE_RPC": "https://rpc.example"},
			origin:      "https://other.example",
			wantNetwork: "Public Global Stellar Network ; September 2015",
			wantRPC:     true,
			wantCORS:    "",
		},
		{
			name:        "testnet passphrase and no rpc",
			env:         map[string]string{"EXNODE_NETWORK": "testnet", "EXNODE_RPC": ""},
			origin:      "https://a.example",
			wantNetwork: "Test SDF Network ; September 2015",
			wantRPC:     false,
			wantCORS:    "*",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("EXNODE_NETWORK", "")
			t.Setenv("EXNODE_CORS", "")
			t.Setenv("EXNODE_RPC", "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			h, err := FromEnv()
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			req.Header.Set("Origin", tc.origin)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			var body struct {
				Network string `json:"network"`
				RPC     bool   `json:"rpc"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != http.StatusOK {
				t.Fatalf("status %d, body %q", rec.Code, rec.Body)
			}
			if body.Network != tc.wantNetwork || body.RPC != tc.wantRPC {
				t.Fatalf("health %+v, want network=%q rpc=%v", body, tc.wantNetwork, tc.wantRPC)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tc.wantCORS {
				t.Fatalf("allow-origin %q, want %q", got, tc.wantCORS)
			}
		})
	}
}
