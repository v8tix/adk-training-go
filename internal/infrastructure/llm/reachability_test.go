package llm

import (
	"net"
	"testing"
	"time"
)

func TestOllamaReachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer ln.Close()

	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{
			name: "reachable server",
			cfg:  Config{OllamaBaseURL: "http://" + ln.Addr().String() + "/v1"},
			want: true,
		},
		{
			name: "nothing listening on that port",
			cfg:  Config{OllamaBaseURL: "http://127.0.0.1:1/v1"},
			want: false,
		},
		{
			name: "malformed URL",
			cfg:  Config{OllamaBaseURL: "://not-a-url"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OllamaReachable(tt.cfg, 500*time.Millisecond); got != tt.want {
				t.Fatalf("OllamaReachable(%q) = %v, want %v", tt.cfg.OllamaBaseURL, got, tt.want)
			}
		})
	}
}
