//nolint:testpackage // Exercises internal request paths and server state conversion.
package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rixlhq/terraform-provider-netcup/internal/provider/scpcommon"
)

func TestActionPaths(t *testing.T) {
	t.Parallel()
	tests := []struct{ action, arg, value, want string }{
		{"snapshot_revert", argSnapshotName, "before-upgrade", "/api/v1/servers/12345/snapshots/before-upgrade/revert"},
		{"snapshot_export", argSnapshotName, "before-upgrade", "/api/v1/servers/12345/snapshots/before-upgrade/export"},
		{"snapshot_delete", argSnapshotName, "before-upgrade", "/api/v1/servers/12345/snapshots/before-upgrade"},
		{"disk_format", "disk_name", "sda", "/api/v1/servers/12345/disks/sda:format"},
		{"firewall_reapply", "mac", "00:11:22:33:44:55", "/api/v1/servers/12345/interfaces/00:11:22:33:44:55/firewall:reapply"},
		{"firewall_restore", "mac", "00:11:22:33:44:55", "/api/v1/servers/12345/interfaces/00:11:22:33:44:55/firewall:restore-copied-policies"},
		{"snapshot_revert", argSnapshotName, "before/10% upgrade", "/api/v1/servers/12345/snapshots/before%2F10%25%20upgrade/revert"},
	}
	for _, tc := range tests {
		t.Run(tc.action+"/"+tc.value, func(t *testing.T) {
			t.Parallel()
			got, err := actionPath(actionSpecs[tc.action], 12345, tc.action, map[string]string{tc.arg: tc.value})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("path = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestActionPathsRejectMissingArguments(t *testing.T) {
	t.Parallel()
	for _, args := range []map[string]string{nil, {argSnapshotName: ""}} {
		if _, err := actionPath(actionSpecs["snapshot_revert"], 12345, "snapshot_revert", args); err == nil {
			t.Errorf("expected missing argument error for %v", args)
		}
	}
}

func TestServerCPUTopologyFromLiveInfo(t *testing.T) {
	t.Parallel()
	for name, live := range map[string]any{
		"present": map[string]any{"sockets": json.Number("2"), "coresPerSocket": json.Number("4")},
		"missing": nil,
		"partial": map[string]any{"sockets": json.Number("2")},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			flat, err := flattenServerJSON(12345, map[string]any{"serverLiveInfo": live})
			if err != nil {
				t.Fatal(err)
			}
			if name != "present" {
				if _, ok := flat["cpu_topology"]; ok {
					t.Fatal("incomplete live info produced CPU topology")
				}
				return
			}
			topology, ok := flat["cpu_topology"].(map[string]any)
			if !ok {
				t.Fatalf("missing CPU topology: %v", flat)
			}
			if topology["socket_count"] != json.Number("2") || topology["cores_per_socket_count"] != json.Number("4") {
				t.Fatalf("unexpected CPU topology: %v", topology)
			}
			schema := serverResourceSchema(context.Background(), &ScpServerResource{})
			if _, err := scpcommon.JSONToTfValue(context.Background(), schema.Type().TerraformType(context.Background()), flat); err != nil {
				t.Fatal(err)
			}
		})
	}
}
