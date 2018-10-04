package routes

import (
	"github.com/go-courier/courier"
	"github.com/go-courier/httptransport/openapi"
	"github.com/pippozq/pushgateway/routes/pushgateway"
)

var VersionRouter = courier.NewRouter(GroupVersion{})
var RootRouter = courier.NewRouter(GroupRoot{})

func init() {
	VersionRouter.Register(pushgateway.MetricsRouter)
	RootRouter.Register(VersionRouter)
	RootRouter.Register(openapi.OpenAPIRouter)
}

type GroupVersion struct {
	courier.EmptyOperator
}

func (g GroupVersion) Path() string {
	return "/v0"
}

type GroupRoot struct {
	courier.EmptyOperator
}

func (root GroupRoot) Path() string {
	return "/pushgateway"
}