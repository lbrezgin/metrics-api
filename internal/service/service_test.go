package service

import (
	"testing"

	"github.com/lbrezgin/telemetry/internal/model"
	"github.com/lbrezgin/telemetry/internal/repository"
	"github.com/lbrezgin/telemetry/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_metricsService_Set(t *testing.T) {
	t.Run("updates value for existing metric", func(t *testing.T) {
		repo := repository.NewMemStorage()
		metSVC := &metricsService{repo: repo}

		repo.Save(testutil.GaugeMetric("GM1", 12.223))
		metSVC.Set("GM1", model.Gauge, 56.12992)
		gm1Updated, err := repo.Find("GM1", model.Gauge)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assert.Equalf(t, 56.12992, *gm1Updated.Value, "Set(): should update value of existing metric")
	})

	t.Run("create new metric if metric doesn't exist", func(t *testing.T) {
		repo := repository.NewMemStorage()
		metSVC := &metricsService{repo: repo}

		metSVC.Set("GM1", model.Gauge, 89.12992)
		gm1Updated, err := repo.Find("GM1", model.Gauge)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assert.Equalf(t, 89.12992, *gm1Updated.Value, "Set(): incorrect value")
		assert.Equalf(t, "GM1", gm1Updated.ID, "Set(): incorrect id")
		assert.Equalf(t, model.Gauge, gm1Updated.MType, "Set(): incorrect metric type")
	})
}

func Test_metricsService_Increment(t *testing.T) {
	t.Run("increments value for existing metric", func(t *testing.T) {
		repo := repository.NewMemStorage()
		metSVC := &metricsService{repo: repo}

		repo.Save(testutil.CounterMetric("CM1", 2))
		metSVC.Increment("CM1", model.Counter, 5)
		cm1Incremented, err := repo.Find("CM1", model.Counter)

		require.NoErrorf(t, err, "unexpected error: %v", err)
		assert.Equalf(t, int64(7), *cm1Incremented.Delta, "Increment(): should increment value of existing metric")
	})

	t.Run("create new metric if metric doesn't exist", func(t *testing.T) {
		repo := repository.NewMemStorage()
		metSVC := &metricsService{repo: repo}

		metSVC.Increment("CM1", model.Counter, 12)
		cm1Incremented, err := repo.Find("CM1", model.Counter)

		require.NoErrorf(t, err, "unexpected error: %v", err)
		assert.Equalf(t, int64(12), *cm1Incremented.Delta, "Increment(): incorrect value")
		assert.Equalf(t, "CM1", cm1Incremented.ID, "Increment(): incorrect id")
		assert.Equalf(t, model.Counter, cm1Incremented.MType, "Increment(): incorrect metric type")
	})
}

func Test_metricsService_GetVal(t *testing.T) {
	type args struct {
		id         string
		metricType string
	}

	type want struct {
		val     string
		errType error
	}

	tests := []struct {
		name string
		repo storage
		args args
		want want
	}{
		{
			name: "returns ErrMetricNotFound if metric doesn't exist",
			repo: repository.NewMemStorage(),
			args: args{
				id:         "Alloc",
				metricType: model.Gauge,
			},
			want: want{
				val:     "",
				errType: repository.ErrMetricNotFound,
			},
		},
		{
			name: "returns ErrUnknownMetricType if given unsupported metric type",
			repo: func() storage {
				r := repository.NewMemStorage()
				r.Save(&model.Metrics{
					ID:    "SM1",
					MType: "Navi",
					Value: testutil.F64ptr(12.12),
				})
				return r
			}(),
			args: args{
				id:         "SM1",
				metricType: "Navi",
			},
			want: want{
				val:     "",
				errType: ErrUnknownMetricType,
			},
		},
		{
			name: "returns ErrGaugeHasNilValue if metric of type gauge has nil value",
			repo: func() storage {
				r := repository.NewMemStorage()
				r.Save(&model.Metrics{
					ID:    "ID12",
					MType: model.Gauge,
					Value: nil,
				})
				return r
			}(),
			args: args{
				id:         "ID12",
				metricType: model.Gauge,
			},
			want: want{
				val:     "",
				errType: ErrGaugeHasNilValue,
			},
		},
		{
			name: "returns ErrCounterHasNilDelta if metric of type counter has nil delta",
			repo: func() storage {
				r := repository.NewMemStorage()
				r.Save(&model.Metrics{
					ID:    "12ID",
					MType: model.Counter,
					Delta: nil,
				})
				return r
			}(),
			args: args{
				id:         "12ID",
				metricType: model.Counter,
			},
			want: want{
				val:     "",
				errType: ErrCounterHasNilDelta,
			},
		},
		{
			name: "returns metric value if metric exists (gauge)",
			repo: func() storage {
				r := repository.NewMemStorage()
				r.Save(testutil.GaugeMetric("G1", 12.355))
				return r
			}(),
			args: args{
				id:         "G1",
				metricType: model.Gauge,
			},
			want: want{
				val:     "12.355",
				errType: nil,
			},
		},
		{
			name: "returns metric delta if metric exists (counter)",
			repo: func() storage {
				r := repository.NewMemStorage()
				r.Save(testutil.CounterMetric("C1", 2))
				return r
			}(),
			args: args{
				id:         "C1",
				metricType: model.Counter,
			},
			want: want{
				val:     "2",
				errType: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &metricsService{
				repo: tt.repo,
			}

			val, err := s.GetVal(tt.args.id, tt.args.metricType)
			// ErrorIs() under the hood compares errors with == if one of
			// them is nil. So ErrorIs(t, nil, nil) will be successful test case.
			assert.ErrorIs(t, err, tt.want.errType)
			assert.Equal(t, tt.want.val, val)
		})
	}
}
