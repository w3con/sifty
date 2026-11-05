package source

import (
	"context"
	"testing"

	"github.com/w3con/sifty/internal/model"
)

// stubSource is a minimal Source implementation for registry tests.
type stubSource struct{ name string }

func (s *stubSource) Discover(_ context.Context, _ string) ([]Ref, error) { return nil, nil }
func (s *stubSource) Resolve(_ context.Context, ref Ref) (model.JobInput, error) {
	j := ref.Partial
	j.Source = s.name
	return j, nil
}

func TestRegistry(t *testing.T) {
	fallback := &stubSource{name: "generic"}
	greenhouse := &stubSource{name: "greenhouse"}
	lever := &stubSource{name: "lever"}

	reg := NewRegistry(fallback)
	reg.Register(`boards-api\.greenhouse\.io`, greenhouse)
	reg.Register(`api\.lever\.co`, lever)

	tests := []struct {
		host string
		want string
	}{
		{"boards-api.greenhouse.io", "greenhouse"},
		{"api.lever.co", "lever"},
		{"api.ashbyhq.com", "generic"},  // no match → fallback
		{"jobs.example.com", "generic"}, // no match → fallback
		{"api.lever.co", "lever"},       // repeat to confirm deterministic
	}
	for _, tc := range tests {
		t.Run(tc.host, func(t *testing.T) {
			s := reg.For(tc.host)
			j, err := s.Resolve(context.Background(), Ref{Source: "test"})
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if j.Source != tc.want {
				t.Errorf("For(%q): got source %q, want %q", tc.host, j.Source, tc.want)
			}
		})
	}
}
