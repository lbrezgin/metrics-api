package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/lbrezgin/telemetry/internal/logger"
	"github.com/lbrezgin/telemetry/internal/model"
	"github.com/lbrezgin/telemetry/internal/repository"
	"github.com/lbrezgin/telemetry/internal/router"
	"github.com/lbrezgin/telemetry/internal/service"
	"github.com/stretchr/testify/assert"
)

// want describes response from server that should
// be returned to the client.
type want struct {
	code        int
	response    string
	contentType string
}

// request contains data for request to the server.
type request struct {
	url         string
	method      string
	contentType string
}

// testCase describes full test data for the rest test case.
// It contains want and request structs respectively.
type testCase struct {
	name    string
	handler *metricsHandler
	request request
	want    want
}

func Test_metricsHandler_Update(t *testing.T) {
	tests := []testCase{
		{
			name: "returns 400 if bad value given (gauge)",
			handler: &metricsHandler{
				svc: service.NewMetricsService(
					repository.NewMemStorage(),
				),
			},
			request: request{
				url:         "/update/gauge/Alloc/plur1bus",
				method:      http.MethodPost,
				contentType: "text/plain",
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "bad value given\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "returns 400 if bad value given (counter)",
			handler: &metricsHandler{
				svc: service.NewMetricsService(
					repository.NewMemStorage(),
				),
			},
			request: request{
				url:         "/update/counter/PollCounter/metkayina",
				method:      http.MethodPost,
				contentType: "text/plain",
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "bad value given\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "returns 405 if request method isn't Post",
			handler: &metricsHandler{
				svc: service.NewMetricsService(
					repository.NewMemStorage(),
				),
			},
			request: request{
				url:         "/update/counter/PollCounter/12",
				method:      http.MethodGet,
				contentType: "text/plain",
			},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name: "returns 400 if metric type isn't supported",
			handler: &metricsHandler{
				svc: service.NewMetricsService(
					repository.NewMemStorage(),
				),
			},
			request: request{
				url:         "/update/bugonia/a/12",
				method:      http.MethodPost,
				contentType: "text/plain",
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "unknown metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "returns 200 if all is set up correct (metric of type gauge)",
			handler: &metricsHandler{
				svc: service.NewMetricsService(
					repository.NewMemStorage(),
				),
			},
			request: request{
				url:         "/update/gauge/StackInuse/12.132",
				method:      http.MethodPost,
				contentType: "text/plain",
			},
			want: want{
				code:        http.StatusOK,
				response:    "",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "returns 200 if all is set up correct (metric of type counter)",
			handler: &metricsHandler{
				svc: service.NewMetricsService(
					repository.NewMemStorage(),
				),
			},
			request: request{
				url:         "/update/counter/PollCount/1",
				method:      http.MethodPost,
				contentType: "text/plain",
			},
			want: want{
				code:        http.StatusOK,
				response:    "",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := router.New(tt.handler, logger.LoggingMiddleware)

			srv := httptest.NewServer(router)
			defer srv.Close()

			req := resty.New().R()
			req.Method = tt.request.method
			req.URL = fmt.Sprintf("%s%s", srv.URL, tt.request.url)
			req.SetHeader("Content-Type", tt.request.contentType)

			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")

			assert.Equal(t, tt.want.code, resp.StatusCode())
			assert.Equal(t, tt.want.response, string(resp.Body()))
			assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))
		})
	}
}

func Test_metricsHandler_Show(t *testing.T) {
	tests := []testCase{
		{
			name: "returns 400 if metric type isn't supported ",
			handler: &metricsHandler{
				svc: service.NewMetricsService(
					repository.NewMemStorage(),
				),
			},
			request: request{
				url:         "/value/unsop/Alloc",
				method:      http.MethodGet,
				contentType: "text/plain",
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "unknown metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "returns 404 if metric not found",
			handler: &metricsHandler{
				svc: service.NewMetricsService(
					repository.NewMemStorage(),
				),
			},
			request: request{
				url:         "/value/gauge/Alloc",
				method:      http.MethodGet,
				contentType: "text/plain",
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "metric not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "returns 200 and metric's value if all is correct (gauge)",
			handler: func() *metricsHandler {
				h := &metricsHandler{
					svc: service.NewMetricsService(
						repository.NewMemStorage(),
					),
				}
				h.svc.Set("G1", model.Gauge, 12.12344)
				return h
			}(),
			request: request{
				url:         "/value/gauge/G1",
				method:      http.MethodGet,
				contentType: "text/plain",
			},
			want: want{
				code:        http.StatusOK,
				response:    "12.12344",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "returns 200 and metric's delta if all is correct (counter)",
			handler: func() *metricsHandler {
				h := &metricsHandler{
					svc: service.NewMetricsService(
						repository.NewMemStorage(),
					),
				}
				h.svc.Increment("C1", model.Counter, 19)
				return h
			}(),
			request: request{
				url:         "/value/counter/C1",
				method:      http.MethodGet,
				contentType: "text/plain",
			},
			want: want{
				code:        http.StatusOK,
				response:    "19",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := router.New(tt.handler, logger.LoggingMiddleware)

			srv := httptest.NewServer(router)
			defer srv.Close()

			req := resty.New().R()
			req.Method = tt.request.method
			req.URL = fmt.Sprintf("%s%s", srv.URL, tt.request.url)
			req.SetHeader("Content-Type", tt.request.contentType)

			resp, err := req.Send()
			assert.NoError(t, err, "error making HTTP request")

			assert.Equal(t, tt.want.code, resp.StatusCode())
			assert.Equal(t, tt.want.response, string(resp.Body()))
			assert.Equal(t, tt.want.contentType, resp.Header().Get("Content-Type"))
		})
	}
}
