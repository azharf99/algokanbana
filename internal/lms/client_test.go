package lms

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type stubTransport struct{}

func (stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		Header:     make(http.Header),
	}, nil
}

func TestSafeReadonlyTransportBlocksMutations(t *testing.T) {
	tr := &SafeReadonlyTransport{Base: stubTransport{}, EnforceGuard: true}

	// Critical POST to LMS or NanoKanbana MUST fail immediately
	reqBlocked, _ := http.NewRequest(http.MethodPost, "https://lms.algonova.id/api/v2/schedule/reschedule", strings.NewReader(`{}`))
	respBlocked, err := tr.RoundTrip(reqBlocked)
	if respBlocked != nil && respBlocked.Body != nil {
		_ = respBlocked.Body.Close()
	}
	if !errors.Is(err, ErrMutatingUpstreamBlocked) {
		t.Fatalf("expected critical POST to be blocked, got err=%v", err)
	}

	reqNKBlocked, _ := http.NewRequest(http.MethodPost, "https://nano-kanbana.vercel.app/id/portal/tutor/taskboard", strings.NewReader(`[]`))
	respNKBlocked, err := tr.RoundTrip(reqNKBlocked)
	if respNKBlocked != nil && respNKBlocked.Body != nil {
		_ = respNKBlocked.Body.Close()
	}
	if !errors.Is(err, ErrMutatingUpstreamBlocked) {
		t.Fatalf("expected Server Action POST to NanoKanbana to be blocked, got err=%v", err)
	}

	// Read-only GET MUST pass
	reqGet, _ := http.NewRequest(http.MethodGet, "https://lms.algonova.id/api/v2/group/index", nil)
	respGet, err := tr.RoundTrip(reqGet)
	if err != nil {
		t.Fatalf("expected GET to be allowed, got err=%v", err)
	}
	if respGet != nil && respGet.Body != nil {
		_ = respGet.Body.Close()
	}
}
