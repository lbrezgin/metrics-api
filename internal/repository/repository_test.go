package repository

import (
	"testing"

	"github.com/lbrezgin/telemetry/internal/model"
	"github.com/lbrezgin/telemetry/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_memStorage_Find(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		metricType string
		want       model.Metrics
		wantErr    bool
	}{
		{
			name:       "returns existing metric",
			id:         "Alloc",
			metricType: model.Gauge,
			want:       *testutil.GaugeMetric("Alloc", 313.23),
			wantErr:    false,
		},
		{
			name:       "returns empty struct and error if metric doesn't exist",
			id:         "BuckHashSys",
			metricType: model.Counter,
			want:       model.Metrics{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &memStorage{
				db: dbMemStorage{},
			}
			alloc := testutil.GaugeMetric("Alloc", 313.23)
			repo.db = append(repo.db, alloc)

			got, gotErr := repo.Find(tt.id, tt.metricType)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Find(): error wasn't expected in this case but got: %v", gotErr)
				}
			} else {
				if tt.wantErr {
					t.Errorf("Find(): error was expected here but got nil")
				}
			}

			if gotErr == nil && !tt.wantErr {
				assert.Equal(t, tt.want, got, "Find(): expected %#v, but got %#v", tt.want, got)
			}
		})
	}
}

// Test_memStorage_Find_returnsCopy checks that memStorage.Find() returns a copy
// of the Metrics struct and that Metrics.Delta/Metrics.Value do not share pointers
// with internal storage. This protects the internal state from mutation.
func Test_memStorage_Find_returnsCopy(t *testing.T) {
	repo := &memStorage{
		db: dbMemStorage{},
	}

	alloc := testutil.GaugeMetric("Alloc", 313.23)
	repo.db = append(repo.db, alloc)

	got, gotErr := repo.Find(alloc.ID, alloc.MType)
	require.Nilf(t, gotErr, "Find(): error wasn't expected in this case but got %v", gotErr)
	require.Equalf(t, *alloc.Value, *got.Value, "precondition: values must be equal before mutation")

	*got.Value = 123.12
	assert.NotEqualf(t, *got.Value, *alloc.Value, `
		Find(): was expected that returned metric can't mutate original object. 
		Find() should return deep copy
	`)
}

func Test_memStorage_Save(t *testing.T) {
	t.Run("successfully saves new metric in the storage", func(t *testing.T) {
		repo := &memStorage{
			db: dbMemStorage{},
		}
		pollCount := testutil.CounterMetric("PollCount", 1)
		repo.Save(pollCount)

		for _, m := range repo.db {
			if m.ID == pollCount.ID && *m.Delta == *pollCount.Delta {
				return
			}
		}
		t.Errorf("Save(): storage doesn't contain saved metric")
	})

	t.Run("if metric with the same ID and Delta/Value exists, it is replaced", func(t *testing.T) {
		repo := &memStorage{
			db: dbMemStorage{},
		}

		nextGC := testutil.GaugeMetric("NextGC", 5.432)
		repo.Save(nextGC)

		nextGC2 := testutil.GaugeMetric("NextGC", 5424.123213)
		repo.Save(nextGC2)

		for _, m := range repo.db {
			if m.ID == nextGC.ID {
				if *m.Value != *nextGC2.Value {
					t.Errorf("Save(): if save existing metric, it data should be updated")
				}
				return
			}
		}
	})
}

func Test_memStorage_List(t *testing.T) {
	newTestRepo := func() *memStorage {
		repo := &memStorage{db: dbMemStorage{}}
		repo.db = append(repo.db, testutil.GaugeMetric("GM1", 1.1))
		repo.db = append(repo.db, testutil.GaugeMetric("GM2", 2.1))
		repo.db = append(repo.db, testutil.GaugeMetric("GM3", 3.1))
		repo.db = append(repo.db, testutil.GaugeMetric("GM4", 4.1))
		repo.db = append(repo.db, testutil.CounterMetric("CM1", 1))
		repo.db = append(repo.db, testutil.CounterMetric("CM2", 2))
		repo.db = append(repo.db, testutil.CounterMetric("CM3", 3))
		repo.db = append(repo.db, testutil.CounterMetric("CM4", 4))
		return repo
	}

	newWant := func() []model.Metrics {
		want := []model.Metrics{}
		want = append(want, *testutil.GaugeMetric("GM1", 1.1))
		want = append(want, *testutil.GaugeMetric("GM2", 2.1))
		want = append(want, *testutil.GaugeMetric("GM3", 3.1))
		want = append(want, *testutil.GaugeMetric("GM4", 4.1))
		want = append(want, *testutil.CounterMetric("CM1", 1))
		want = append(want, *testutil.CounterMetric("CM2", 2))
		want = append(want, *testutil.CounterMetric("CM3", 3))
		want = append(want, *testutil.CounterMetric("CM4", 4))
		return want
	}

	t.Run("returns all metrics from the repository", func(t *testing.T) {
		repo := newTestRepo()
		want := newWant()
		got := repo.List()

		assert.Equalf(t, want, got, "List(): expected %v, got %v", want, got)
	})

	t.Run("returned metrics should be copies", func(t *testing.T) {
		repo := newTestRepo()
		want := newWant()
		got := repo.List()

		for i := range got {
			assert.NotSamef(t, &got[i], &want[i], "List(): expected copy but got the original")
			if got[i].Delta != nil {
				assert.NotSamef(t, got[i].Delta, want[i].Delta, "List(): expected copy pointer of the Delta, but got original")
			}
			if got[i].Value != nil {
				assert.NotSamef(t, got[i].Value, want[i].Value, "List(): expected copy pointer of the Value, but got original")
			}
		}
	})
}

func Test_memStorage_cloneMetric(t *testing.T) {
	tm1 := testutil.GaugeMetric("TM1", 12.23)
	tm1Clone := cloneMetric(tm1)

	assert.NotSamef(t, tm1, &tm1Clone, "cloneMetric(): should return copy")
	assert.NotSamef(t, tm1.Value, tm1Clone.Value, "cloneMetric(): pointer values also should be copied")
}
