package provider_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScpServerAction_basic(t *testing.T) {
	server := newServerActionMockServer(t)
	defer server.Close()

	providerConfig := fmt.Sprintf(`
provider "netcup" {
  scp_access_token = "mock-token"
  scp_base_url     = "%s/scp-core"
}
`, server.URL)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "netcup_scp_server_action" "test" {
  server_id = 12345
  action    = "start"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_action.test", "server_id", "12345"),
					resource.TestCheckResourceAttr("netcup_scp_server_action.test", "action", "start"),
					resource.TestCheckResourceAttrSet("netcup_scp_server_action.test", "id"),
				),
			},
			{
				Config: providerConfig + `
resource "netcup_scp_server_action" "test" {
  server_id = 12345
  action    = "start"
  triggers  = { run = "second" }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_server_action.test", "triggers.run", "second"),
					resource.TestCheckResourceAttrSet("netcup_scp_server_action.test", "id"),
				),
			},
		},
	})
}

// TestAccScpServerAction_bodyless verifies POST and DELETE actions keep an empty
// payload while sending the exact JSON media type required by the SCP API.
func TestAccScpServerAction_bodyless(t *testing.T) {
	for _, action := range []string{"rescue_activate", "rescue_deactivate", "iso_detach"} {
		t.Run(action, func(t *testing.T) {
			server := newServerActionMockServer(t)
			defer server.Close()
			config := fmt.Sprintf(`
provider "netcup" {
 scp_access_token = "mock-token"
 scp_base_url = "%s/scp-core"
}
resource "netcup_scp_server_action" "test" {
 server_id = 12345
 action = %q
}
`, server.URL, action)
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("netcup_scp_server_action.test", "action", action),
						resource.TestCheckResourceAttrSet("netcup_scp_server_action.test", "id"),
					),
				}},
			})
		})
	}
}

type serverActionMock struct {
	mu      sync.Mutex
	actions []string
}

func (m *serverActionMock) handleAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	m.mu.Lock()
	m.actions = append(m.actions, fmt.Sprintf("%s: %v", strings.ToLower(r.URL.Query().Get("stateOption")), payload["state"]))
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{"uuid": "task-action-1", "state": taskStateFinished}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleBodyless models the media type requirement without accepting a JSON
// object in place of the empty request body.
func (m *serverActionMock) handleBodyless(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"error.content.type.invalid","message":"The HTTP header Content-Type must be 'type/subtype'"}`))
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) != 0 {
		http.Error(w, "expected an empty request body", http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	m.actions = append(m.actions, r.Method+" "+r.URL.Path)
	m.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{"uuid": "task-bodyless-1", "state": taskStateFinished}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newServerActionMockServer(t *testing.T) *httptest.Server {
	t.Helper()

	mock := &serverActionMock{}
	mux := http.NewServeMux()
	mux.HandleFunc("/scp-core/api/v1/servers/12345", mock.handleAction)
	mux.HandleFunc("POST /scp-core/api/v1/servers/12345/rescuesystem", mock.handleBodyless)
	mux.HandleFunc("DELETE /scp-core/api/v1/servers/12345/rescuesystem", mock.handleBodyless)
	mux.HandleFunc("DELETE /scp-core/api/v1/servers/12345/iso", mock.handleBodyless)

	return httptest.NewServer(mux)
}
