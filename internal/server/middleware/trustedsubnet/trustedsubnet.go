// Package trustedsubnet Проверка, что переданный в заголовке запроса X-Real-IP IP-адрес агента входит в доверенную подсеть
package trustedsubnet

import (
	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
	"go.uber.org/zap"
	"net"
	"net/http"
)

func TrustedSubnetMiddleware(trustedsubnet *net.IPNet) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			if trustedsubnet == nil {
				next.ServeHTTP(w, r)
				return
			}
			realIP := r.Header.Get("X-Real-IP")
			if realIP == "" {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("empty Header X-Real-IP"))
				return
			}

			ip := net.ParseIP(realIP)
			if ip == nil {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("Invalid IP address in Header X-Real-IP"))
				return
			}

			if !trustedsubnet.Contains(ip) {

				logger.Log.Infow("IP address not in allowed in this subnet",
					zap.String("ip", realIP),
					zap.String("subnet", trustedsubnet.String()))

				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("IP address not in allowed in this subnet"))
				return

			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
