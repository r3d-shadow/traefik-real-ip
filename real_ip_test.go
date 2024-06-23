package traefik_real_ip_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	plugin "github.com/r3d-shadow/traefik-real-ip"
)

func TestNew(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.ForwardedForDepth = 1 // Set the depth based on your test requirements

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := plugin.New(ctx, next, cfg, "traefik-real-ip")
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		desc          string
		xForwardedFor string
		expected      string
	}{
		{
			desc:          "don't forward when depth is 1",
			xForwardedFor: "10.0.0.1, 192.168.1.1",
			expected:      "10.0.0.1",
		},
		{
			desc:          "forward when depth is 2",
			xForwardedFor: "10.0.0.1, 192.168.1.1",
			expected:      "192.168.1.1",
		},
		{
			desc:          "fallback to remote address when no X-Forwarded-For",
			xForwardedFor: "",
			expected:      "127.0.0.1", // Assuming localhost default address
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("X-Forwarded-For", test.xForwardedFor)

			handler.ServeHTTP(recorder, req)

			assertHeader(t, req, "X-Real-Ip", test.expected)
		})
	}
}

func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()

	if req.Header.Get(key) != expected {
		t.Errorf("invalid header value: %s, expected: %s", req.Header.Get(key), expected)
	}
}
