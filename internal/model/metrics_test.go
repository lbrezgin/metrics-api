package model

import (
	"testing"
)

func TestMetrics_Validate(t *testing.T) {
	type fields struct {
		ID    string
		MType string
		Delta *int64
		Value *float64
		Hash  string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "return nil if metric is valid",
			fields: fields{
				ID:    "Vecna",
				MType: Gauge,
				Delta: nil,
				Value: f64ptr(12.123),
			},
			wantErr: false,
		},
		{
			name: "return err if has invalid ID",
			fields: fields{
				ID:    "",
				MType: Gauge,
				Delta: nil,
				Value: f64ptr(12.123),
			},
			wantErr: true,
		},
		{
			name: "return err if has invalid MType",
			fields: fields{
				ID:    "Vecna",
				MType: "11",
				Delta: nil,
				Value: f64ptr(12.123),
			},
			wantErr: true,
		},
		{
			name: "return err if has invalid Delta",
			fields: fields{
				ID:    "Vecna",
				MType: Counter,
				Delta: nil,
				Value: nil,
			},
			wantErr: true,
		},
		{
			name: "return err if has invalid Value",
			fields: fields{
				ID:    "Vecna",
				MType: Gauge,
				Delta: nil,
				Value: nil,
			},
			wantErr: true,
		},
		{
			name: "return err if Counter has Value",
			fields: fields{
				ID:    "Vecna",
				MType: Counter,
				Delta: i64ptr(1),
				Value: f64ptr(12.1),
			},
			wantErr: true,
		},
		{
			name: "return err if Gauge has Delta",
			fields: fields{
				ID:    "Vecna",
				MType: Gauge,
				Delta: i64ptr(1),
				Value: f64ptr(12.1),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metrics{
				ID:    tt.fields.ID,
				MType: tt.fields.MType,
				Delta: tt.fields.Delta,
				Value: tt.fields.Value,
				Hash:  tt.fields.Hash,
			}
			if err := m.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func f64ptr(v float64) *float64 {
	return &v
}

func i64ptr(d int64) *int64 {
	return &d
}
