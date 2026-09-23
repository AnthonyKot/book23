package ledger

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type response struct {
	status int
	body   string
}

func request(t *testing.T, handler http.Handler, host, token, path string) response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "http://"+host+path, nil)
	req.Host = host
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	result := recorder.Result()
	defer result.Body.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response{status: result.StatusCode, body: string(body)}
}

func TestChapter01ExerciseCases(t *testing.T) {
	type expectation struct {
		status  int
		include string
		exclude string
	}
	tests := []struct {
		name, token, path string
		vulnerable, fixed expectation
	}{
		{"v2 Alice own invoice", "alice-token", "/v2/invoices/104", expectation{200, `"tenant":"Cedar"`, ""}, expectation{200, `"tenant":"Cedar"`, ""}},
		{"v2 Alice Birch invoice", "alice-token", "/v2/invoices/205", expectation{200, `"tenant":"Birch"`, ""}, expectation{404, "", "Birch"}},
		{"v2 Ben Cedar invoice", "ben-token", "/v2/invoices/104", expectation{200, `"tenant":"Cedar"`, ""}, expectation{404, "", "Cedar"}},
		{"v1 Alice own invoice", "alice-token", "/v1/invoices/104", expectation{200, `"tenant":"Cedar"`, ""}, expectation{200, `"tenant":"Cedar"`, ""}},
		{"v1 Alice Birch invoice", "alice-token", "/v1/invoices/205", expectation{200, `"tenant":"Birch"`, ""}, expectation{404, "", "Birch"}},
		{"v1 Ben Cedar invoice", "ben-token", "/v1/invoices/104", expectation{200, `"tenant":"Cedar"`, ""}, expectation{404, "", "Cedar"}},
		{"PDF Alice own invoice", "alice-token", "/v1/invoices/104/pdf", expectation{200, "tenant=Cedar", ""}, expectation{200, "tenant=Cedar", ""}},
		{"PDF Alice Birch invoice", "alice-token", "/v1/invoices/205/pdf", expectation{200, "tenant=Birch", ""}, expectation{404, "", "Birch"}},
		{"PDF Ben Cedar invoice", "ben-token", "/v1/invoices/104/pdf", expectation{200, "tenant=Cedar", ""}, expectation{404, "", "Cedar"}},
		{"safe list decoy", "alice-token", "/v2/me/invoices", expectation{200, `"id":104`, `"id":205`}, expectation{200, `"id":104`, `"id":205`}},
		{"Dana Cedar admin", "dana-token", "/v2/invoices/104", expectation{200, `"tenant":"Cedar"`, ""}, expectation{200, `"tenant":"Cedar"`, ""}},
	}

	apps := map[Mode]*App{Vulnerable: NewApp(Vulnerable), Fixed: NewApp(Fixed)}
	for _, test := range tests {
		for _, mode := range []Mode{Vulnerable, Fixed} {
			t.Run(string(mode)+"/"+test.name, func(t *testing.T) {
				want := test.vulnerable
				if mode == Fixed {
					want = test.fixed
				}
				got := request(t, apps[mode], PublicHost, test.token, test.path)
				if got.status != want.status {
					t.Fatalf("status = %d, want %d; body=%q", got.status, want.status, got.body)
				}
				if want.include != "" && !strings.Contains(got.body, want.include) {
					t.Errorf("body %q does not include %q", got.body, want.include)
				}
				if want.exclude != "" && strings.Contains(got.body, want.exclude) {
					t.Errorf("body %q unexpectedly includes %q", got.body, want.exclude)
				}
			})
		}
	}
}
