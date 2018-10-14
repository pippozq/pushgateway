package pushgateway

import (
	"context"
	"github.com/go-courier/courier"
	"github.com/go-courier/httptransport/httpx"
	"github.com/pippozq/pushgateway/global"
	"github.com/pippozq/pushgateway/modules/pushgateway"
)

var MetricsRouter = courier.NewRouter(Metrics{})

func init() {
	MetricsRouter.Register(courier.NewRouter(GetMetrics{}))
	MetricsRouter.Register(courier.NewRouter(PushMetrics{}))
}

type Metrics struct {
	courier.EmptyOperator
}

func (c Metrics) Path() string {
	return "/metrics"
}

// Get Metrics
type GetMetrics struct {
	httpx.MethodGet
}

func (req GetMetrics) Output(c context.Context) (resp interface{}, err error) {
	return pushgateway.NewPushGateWayController(nil, global.Config.RedisAgent).GetMetrics()
}

// Push Metrics
type PushMetrics struct {
	httpx.MethodPost
	Body pushgateway.PushData `in:"body"`
}

func (req PushMetrics) Output(c context.Context) (resp interface{}, err error) {

	pushCtrl := pushgateway.NewPushGateWayController(&req.Body, global.Config.RedisAgent)

	return pushCtrl.CacheMetric()
}
