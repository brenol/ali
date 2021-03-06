package gui

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"go.uber.org/atomic"

	"github.com/nakabonne/ali/attacker"
)

func TestRedrawChart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name             string
		results          []*attacker.Result
		latencyChart     LineChart
		percentilesChart LineChart
	}{
		{
			name: "one result received",
			results: []*attacker.Result{
				{
					Latency: 1000000,
					P50:     500000,
					P90:     900000,
					P95:     950000,
					P99:     990000,
				},
			},
			latencyChart: func() LineChart {
				l := NewMockLineChart(ctrl)
				l.EXPECT().Series("latency", []float64{1.0}, gomock.Any())
				return l
			}(),
			percentilesChart: func() LineChart {
				l := NewMockLineChart(ctrl)
				gomock.InOrder(
					l.EXPECT().Series("p50", []float64{0.5}, gomock.Any()),
					l.EXPECT().Series("p90", []float64{0.9}, gomock.Any()),
					l.EXPECT().Series("p95", []float64{0.95}, gomock.Any()),
					l.EXPECT().Series("p99", []float64{0.99}, gomock.Any()),
				)
				return l
			}(),
		},
		{
			name: "two results received",
			results: []*attacker.Result{
				{
					Latency: 1000000,
					P50:     500000,
					P90:     900000,
					P95:     950000,
					P99:     990000,
				},
				{
					Latency: 2000000,
					P50:     1000000,
					P90:     1800000,
					P95:     1900000,
					P99:     1980000,
				},
			},
			latencyChart: func() LineChart {
				l := NewMockLineChart(ctrl)
				l.EXPECT().Series("latency", []float64{1.0}, gomock.Any())
				l.EXPECT().Series("latency", []float64{1.0, 2.0}, gomock.Any())
				return l
			}(),
			percentilesChart: func() LineChart {
				l := NewMockLineChart(ctrl)

				gomock.InOrder(l.EXPECT().Series("p50", []float64{0.5}, gomock.Any()),
					l.EXPECT().Series("p90", []float64{0.9}, gomock.Any()),
					l.EXPECT().Series("p95", []float64{0.95}, gomock.Any()),
					l.EXPECT().Series("p99", []float64{0.99}, gomock.Any()),
					l.EXPECT().Series("p50", []float64{0.5, 1.0}, gomock.Any()),
					l.EXPECT().Series("p90", []float64{0.9, 1.8}, gomock.Any()),
					l.EXPECT().Series("p95", []float64{0.95, 1.9}, gomock.Any()),
					l.EXPECT().Series("p99", []float64{0.99, 1.98}, gomock.Any()),
				)
				return l
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &drawer{
				widgets:      &widgets{latencyChart: tt.latencyChart, percentilesChart: tt.percentilesChart},
				chartCh:      make(chan *attacker.Result),
				gaugeCh:      make(chan bool, 100),
				chartDrawing: atomic.NewBool(false),
			}
			go d.redrawChart(ctx, len(tt.results))
			for _, res := range tt.results {
				d.chartCh <- res
			}
			d.chartCh <- &attacker.Result{End: true}
		})
	}
}

func TestRedrawGauge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name  string
		size  int
		count int
		gauge Gauge
	}{
		{
			name: "draw once",
			size: 1,
			gauge: func() Gauge {
				g := NewMockGauge(ctrl)
				g.EXPECT().Percent(0)
				g.EXPECT().Percent(100)
				return g
			}(),
		},
		{
			name: "draw twice",
			size: 2,
			gauge: func() Gauge {
				g := NewMockGauge(ctrl)
				g.EXPECT().Percent(0)
				g.EXPECT().Percent(50)
				g.EXPECT().Percent(100)
				return g
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &drawer{
				widgets:      &widgets{progressGauge: tt.gauge},
				gaugeCh:      make(chan bool),
				chartDrawing: atomic.NewBool(false),
			}
			go d.redrawGauge(ctx, tt.size)
			for i := 0; i < tt.size; i++ {
				d.gaugeCh <- false
			}
			d.gaugeCh <- true
		})
	}
}

func TestRedrawMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name            string
		metrics         *attacker.Metrics
		latenciesText   Text
		bytesText       Text
		othersText      Text
		statusCodesText Text
		errorsText      Text
	}{
		{
			name: "nil metrics given",
			latenciesText: func() Text {
				t := NewMockText(ctrl)
				return t
			}(),
			bytesText: func() Text {
				t := NewMockText(ctrl)
				return t
			}(),
			othersText: func() Text {
				t := NewMockText(ctrl)
				return t
			}(),
			statusCodesText: func() Text {
				t := NewMockText(ctrl)
				return t
			}(),
			errorsText: func() Text {
				t := NewMockText(ctrl)
				return t
			}(),
		},
		{
			name: "with errors",
			metrics: &attacker.Metrics{
				Latencies: attacker.LatencyMetrics{
					Total: 1,
					Mean:  1,
					P50:   1,
					P90:   1,
					P95:   1,
					P99:   1,
					Max:   1,
					Min:   1,
				},
				BytesIn: attacker.ByteMetrics{
					Total: 1,
					Mean:  1,
				},
				BytesOut: attacker.ByteMetrics{
					Total: 1,
					Mean:  1,
				},
				Earliest:    time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Latest:      time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				End:         time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				Duration:    1,
				Wait:        1,
				Requests:    1,
				Rate:        1,
				Throughput:  1,
				Success:     1,
				StatusCodes: map[string]int{"200": 2},
				Errors:      []string{"error1"},
			},
			latenciesText: func() Text {
				t := NewMockText(ctrl)
				t.EXPECT().Write(`Total: 1ns
Mean: 1ns
P50: 1ns
P90: 1ns
P95: 1ns
P99: 1ns
Max: 1ns
Min: 1ns`, gomock.Any())
				return t
			}(),

			bytesText: func() Text {
				t := NewMockText(ctrl)
				t.EXPECT().Write(`In:
  Total: 1
  Mean: 1
Out:
  Total: 1
  Mean: 1`, gomock.Any())
				return t
			}(),

			statusCodesText: func() Text {
				t := NewMockText(ctrl)
				t.EXPECT().Write(`"200": 2
`, gomock.Any())
				return t
			}(),

			errorsText: func() Text {
				t := NewMockText(ctrl)
				t.EXPECT().Write(`- error1
`, gomock.Any())
				return t
			}(),

			othersText: func() Text {
				t := NewMockText(ctrl)
				t.EXPECT().Write(`Duration: 1ns
Wait: 1ns
Requests: 1
Rate: 1.000000
Throughput: 1.000000
Success: 1.000000
Earliest: 2009-11-10T23:00:00Z
Latest: 2009-11-10T23:00:00Z
End: 2009-11-10T23:00:00Z`, gomock.Any())

				return t
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			d := &drawer{
				widgets: &widgets{
					latenciesText:   tt.latenciesText,
					bytesText:       tt.bytesText,
					othersText:      tt.othersText,
					statusCodesText: tt.statusCodesText,
					errorsText:      tt.errorsText,
				},
				metricsCh: make(chan *attacker.Metrics),
			}
			go d.redrawMetrics(ctx)
			d.metricsCh <- tt.metrics
			cancel()
			// TODO: Stop waiting inappropriately.
			// Currently waiting in a rough manner to ensure that the mock function
			// is called in the `redrawMetrics`, but it is unstable and inefficient.
			time.Sleep(2 * time.Second)
		})
	}
}
