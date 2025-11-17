package handlers

import (
	"crypto/rsa"
	"github.com/go-chi/chi/v5"
	_ "github.com/s-turchinskiy/metrics/internal/server/handlers/swagger"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/gzip"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/hash"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	rsamiddleware "github.com/s-turchinskiy/metrics/internal/server/middleware/rsa"
	"github.com/s-turchinskiy/metrics/internal/server/middleware/trustedsubnet"
	httpswagger "github.com/swaggo/http-swagger"
	"golang.org/x/exp/slices"
	"net"
	"net/http"
	"net/http/pprof"
)

// @Title MetricStorage API
// @Description Сервис хранения метрик.
// @Version 1.0

// @Contact.email s.turchinskiy@yandex.ru

// @BasePath /
// @Host nohost.io:8080

// @SecurityDefinitions.apikey ApiKeyAuth
// @In header
// @Name authorization

// @Tag.name Info
// @Tag.description "Группа запросов метрик"

// @Tag.name Update
// @Tag.description "Группа обновления метрик"

// @Tag.name Ping
// @Tag.description "Группа проверки работоспособности сервиса"

type filterType map[string]map[string][]string
type middlewareType func(next http.Handler) http.Handler

func Router(h *MetricsHandler, rsaPrivateKey *rsa.PrivateKey, hashKey string, trustedSubnet *net.IPNet) chi.Router {

	filter := make(map[string]map[string][]string, 1)
	filterRSA := make(map[string][]string, 1)
	filterRSA["/update"] = []string{http.MethodPost}
	filter["RSA"] = filterRSA

	router := chi.NewRouter()
	router.Use(hash.HashWriteMiddleware(hashKey))
	router.Use(hash.HashReadMiddleware(hashKey))
	router.Use(trustedsubnet.TrustedSubnetMiddleware(trustedSubnet))
	router.Use(filteringMiddleware(filter, "RSA", rsamiddleware.RSADecrypt(rsaPrivateKey)))
	router.Use(gzip.GzipMiddleware)
	router.Use(logger.Logger)
	router.Route("/update", func(r chi.Router) {
		r.Post("/", h.UpdateMetricJSON)
		r.Get("/{MetricsType}/{MetricsName}/{MetricsValue}", h.UpdateMetric)
	})
	router.Route("/updates", func(r chi.Router) {
		r.Post("/", h.UpdateMetricsBatch)
	})
	router.Route("/value", func(r chi.Router) {
		r.Post("/", h.GetTypedMetric)
		r.Get("/{MetricsType}/{MetricsName}", h.GetMetric)
	})
	router.Route("/ping", func(r chi.Router) {
		r.Get("/", h.Ping)
	})

	router.Get(`/`, h.GetAllMetrics)
	router.Mount("/swagger", httpswagger.WrapHandler)

	router.Route("/debug/pprof", func(r chi.Router) {
		r.Get("/", pprof.Index)
		r.Get("/profile", pprof.Profile)
		r.Get("/trace", pprof.Trace)
		r.Get("/symbol", pprof.Symbol)
		r.Get("/cmdline", pprof.Cmdline)

		r.Get("/goroutine", pprof.Handler("goroutine").ServeHTTP)
		r.Get("/heap", pprof.Handler("heap").ServeHTTP)
		r.Get("/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
		r.Get("/block", pprof.Handler("block").ServeHTTP)
		r.Get("/allocs", pprof.Handler("allocs").ServeHTTP)
		r.Get("/mutex", pprof.Handler("mutex").ServeHTTP)

	})

	return router

}

func filteredMiddleware(filter map[string]map[string][]string, nameMiddleware, RequestURI, Method string) bool {

	filterURI, exist := filter[nameMiddleware]
	if !exist {
		return false
	}

	methods, exist := filterURI[RequestURI]
	if !exist {
		return false
	}

	if !slices.Contains(methods, Method) {
		return false
	}

	return true
}

func filteringMiddleware(filter filterType, nameMiddleware string,
	middleware middlewareType) middlewareType {

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			if filteredMiddleware(filter, nameMiddleware, r.RequestURI, r.Method) {
				middleware(next)
			}
		}

		return http.HandlerFunc(fn)
	}
}
