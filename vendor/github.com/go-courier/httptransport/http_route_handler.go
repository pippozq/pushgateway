package httptransport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/go-courier/courier"
	"github.com/go-courier/httptransport/httpx"
	"github.com/go-courier/httptransport/transformers"
	"github.com/go-courier/reflectx/typesutil"
	"github.com/go-courier/statuserror"
)

func NewHttpRouteHandler(serviceMeta *ServiceMeta, httpRoute *HttpRouteMeta, requestTransformerMgr *RequestTransformerMgr) *HttpRouteHandler {
	operatorFactories := httpRoute.OperatorFactories()
	if len(operatorFactories) == 0 {
		panic(fmt.Errorf("missing valid operator"))
	}

	requestTransformers := make([]*RequestTransformer, len(operatorFactories))
	for i := range operatorFactories {
		opFactory := operatorFactories[i]
		rt, err := requestTransformerMgr.NewRequestTransformer(opFactory.Type)
		if err != nil {
			panic(err)
		}
		requestTransformers[i] = rt
	}

	return &HttpRouteHandler{
		RequestTransformerMgr: requestTransformerMgr,
		HttpRouteMeta:         httpRoute,

		serviceMeta:         serviceMeta,
		operatorFactories:   operatorFactories,
		requestTransformers: requestTransformers,
	}
}

type HttpRouteHandler struct {
	*RequestTransformerMgr
	*HttpRouteMeta

	serviceMeta         *ServiceMeta
	operatorFactories   []*courier.OperatorMeta
	requestTransformers []*RequestTransformer
}

func (handler *HttpRouteHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx = ContextWithHttpRequest(ctx, r)
	ctx = ContextWithServiceMeta(ctx, *handler.serviceMeta)

	rw.Header().Set("X-Service", handler.serviceMeta.String())

	requestInfo := NewRequestInfo(r)

	for i := range handler.operatorFactories {
		opFactory := handler.operatorFactories[i]

		op := opFactory.New()

		rt := handler.requestTransformers[i]
		if rt != nil {
			err := rt.DecodeFromRequestInfo(requestInfo, op)
			if err != nil {
				handler.writeErr(rw, r, err)
				return
			}
		}

		result, err := op.Output(ctx)

		if err != nil {
			handler.writeErr(rw, r, err)
			return
		}

		if !opFactory.IsLast {
			// set result in context with key of operator name
			ctx = context.WithValue(ctx, opFactory.ContextKey, result)
			continue
		}

		handler.writeResp(rw, r, result)
	}
}

func (handler *HttpRouteHandler) writeResp(rw http.ResponseWriter, r *http.Request, resp interface{}) {
	if err := httpx.ResponseFrom(resp).WriteTo(rw, r, func(w io.Writer, response *httpx.Response) error {
		transformer, err := handler.TransformerMgr.NewTransformer(typesutil.FromRType(reflect.TypeOf(response.Value)), transformers.TransformerOption{
			MIME: response.ContentType,
		})
		if err != nil {
			return err
		}
		mediaType, err := transformer.EncodeToWriter(w, response.Value)
		if err != nil {
			return err
		}
		response.ContentType = mediaType
		return nil
	}); err != nil {
		handler.writeErr(rw, r, err)
	}
}

func (handler *HttpRouteHandler) writeErr(rw http.ResponseWriter, r *http.Request, err error) {
	if _, ok := err.(httpx.RedirectDescriber); !ok {
		err = statuserror.FromErr(err).AppendSource(handler.serviceMeta.String())
	}
	handler.writeResp(rw, r, err)
}
