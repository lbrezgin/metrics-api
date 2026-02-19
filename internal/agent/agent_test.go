package agent

import (
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/lbrezgin/telemetry/internal/agent/stats"
	"github.com/lbrezgin/telemetry/internal/config"
	"github.com/lbrezgin/telemetry/internal/model"
	"github.com/stretchr/testify/assert"
)

func Test_agent_sendMetrics(t *testing.T) {

	tests := []struct {
		name    string
		agent   *agent
		metrics stats.StatsStorage
		wantErr bool
	}{
		{
			name: "skip metric if it has unsupported type and doesn't return error",
			metrics: stats.StatsStorage{
				"Alloc": stats.Stat{Type: "random type", IntVal: 12},
			},
			wantErr: false,
		},
		{
			name: "doesn't return error if all arguments are correct (gauge)",
			metrics: stats.StatsStorage{
				"Alloc": stats.Stat{Type: model.Gauge, FltVal: 19.99},
			},
			wantErr: false,
		},
		{
			name: "doesn't return error if all arguments are correct (counter)",
			metrics: stats.StatsStorage{
				"Alloc": stats.Stat{Type: model.Counter, IntVal: 112},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &agent{
				stats:  stats.NewRuntimeStats(),
				client: resty.New(),
				cfg: &config.Agent{
					PollInterval:   2,
					ReportInterval: 10,
				},
			}

			gotErr := a.sendMetrics(tt.metrics)
			if tt.wantErr {
				assert.NotNil(t, gotErr)
			}
		})
	}
}
