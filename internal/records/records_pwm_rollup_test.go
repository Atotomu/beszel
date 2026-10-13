//go:build testing

package records_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/henrygd/beszel/internal/entities/system"
	"github.com/henrygd/beszel/internal/records"
	"github.com/henrygd/beszel/internal/tests"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/stretchr/testify/require"
)

func TestPWMHistoryRollup(t *testing.T) {
	for _, tier := range []struct {
		shorter, longer string
		count           int
	}{
		{"1m", "10m", 9},
		{"10m", "20m", 2},
		{"20m", "120m", 6},
		{"120m", "480m", 4},
	} {
		t.Run(tier.longer, func(t *testing.T) {
			hub, err := tests.NewTestHub(t.TempDir())
			require.NoError(t, err)
			defer hub.Cleanup()
			user, err := tests.CreateUser(hub, "pwm@example.com", "testtesttest")
			require.NoError(t, err)
			sys, err := tests.CreateRecord(hub, "systems", map[string]any{
				"name": "pwm-system", "host": "localhost", "port": "45876",
				"status": "up", "users": []string{user.Id},
			})
			require.NoError(t, err)
			for i := range tier.count + 1 {
				stats := system.Stats{}
				if i < tier.count {
					stats.PWM = map[string]float64{"pwm1": 50, "stopped": 0}
					if i < 2 {
						stats.PWM["pwm1"] = float64(i * 100)
						stats.PWM["intermittent"] = float64(i * 100)
					}
				}
				_, err := tests.CreateRecord(hub, "system_stats", map[string]any{
					"system": sys.Id, "type": tier.shorter, "stats": stats,
					"created": time.Now().UTC().Add(-time.Minute).Format(types.DefaultDateLayout),
				})
				require.NoError(t, err)
			}
			records.NewRecordManager(hub).CreateLongerRecords()
			longer, err := hub.FindAllRecords("system_stats", dbx.HashExp{"system": sys.Id, "type": tier.longer})
			require.NoError(t, err)
			require.Len(t, longer, 1)
			var stats system.Stats
			require.NoError(t, json.Unmarshal(longer[0].GetRaw("stats").(types.JSONRaw), &stats))
			require.Equal(t, map[string]float64{"pwm1": 50, "intermittent": 50, "stopped": 0}, stats.PWM)
		})
	}
}
