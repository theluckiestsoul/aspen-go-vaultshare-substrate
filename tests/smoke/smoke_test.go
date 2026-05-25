package smoke_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/micro1/aspen-go-vaultshare-substrate"
)

func TestMemberCanUseOwnReportWorkflow(t *testing.T) {
	handler := vaultshare.NewServer()

	client := memberClient("alice", "blue")

	resp := do(t, handler, client, http.MethodGet, "/reports/rpt-alice", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected own report detail to be readable, got %d", resp.StatusCode)
	}

	resp = do(t, handler, client, http.MethodPost, "/reports/rpt-alice/clone", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected own report clone to succeed, got %d", resp.StatusCode)
	}
	var cloned struct {
		ID string `json:"id"`
	}
	decode(t, resp, &cloned)
	if cloned.ID == "" {
		t.Fatal("expected clone response to include an id")
	}

	resp = do(t, handler, client, http.MethodPatch, "/reports/"+cloned.ID, strings.NewReader(`{"title":"renamed"}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected owner patch to succeed, got %d", resp.StatusCode)
	}

	resp = do(t, handler, client, http.MethodPost, "/reports/"+cloned.ID+"/share", strings.NewReader(`{"team_id":"green"}`))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected owner share to succeed, got %d", resp.StatusCode)
	}

	resp = do(t, handler, client, http.MethodGet, "/reports/"+cloned.ID+"/export", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected owner export to succeed, got %d", resp.StatusCode)
	}
}

func TestMemberCanReadPublicAttachment(t *testing.T) {
	handler := vaultshare.NewServer()

	resp := do(t, handler, memberClient("carol", "green"), http.MethodGet, "/reports/rpt-public/attachments/public_readme.md", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected public attachment to be readable, got %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "Public launch checklist") {
		t.Fatalf("expected public attachment body, got %q", body)
	}
}

func TestAdminCanInspectFullReportData(t *testing.T) {
	handler := vaultshare.NewServer()

	resp := do(t, handler, adminClient(), http.MethodGet, "/reports", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected admin report list to succeed, got %d", resp.StatusCode)
	}

	body := readBody(t, resp)
	if !strings.Contains(body, "PRIVATE-EXPORT-TOKEN-alice") {
		t.Fatalf("expected admin list to include private export tokens, got %q", body)
	}
}

func TestOwnerCanDownloadOwnNormalExportArtifacts(t *testing.T) {
	handler := vaultshare.NewServer()

	resp := do(t, handler, memberClient("alice", "blue"), http.MethodGet, "/exports/run-alice-normal/artifacts", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected owner to read normal export artifacts, got %d", resp.StatusCode)
	}
	body := readBody(t, resp)
	if !strings.Contains(body, "normal alice export") {
		t.Fatalf("expected normal export artifact content, got %q", body)
	}
}

type headerClient struct {
	headers map[string]string
}

func memberClient(userID string, teams string) headerClient {
	return headerClient{
		headers: map[string]string{
			"X-Role":     "member",
			"X-User-ID":  userID,
			"X-Team-IDs": teams,
		},
	}
}

func adminClient() headerClient {
	return headerClient{
		headers: map[string]string{
			"X-Role":    "admin",
			"X-User-ID": "admin",
		},
	}
}

func do(t *testing.T, handler http.Handler, c headerClient, method string, target string, body io.Reader) *http.Response {
	t.Helper()
	if body == nil {
		body = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Result()
}

func decode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
