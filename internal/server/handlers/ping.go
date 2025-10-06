package handlers

import (
	"net/http"

	"github.com/s-turchinskiy/metrics/internal/server/middleware/logger"
)

// Ping godoc
// @Tags Ping
// @Summary пинг сервиса
// @ID pingPing
// @Accept  json
// @Produce html
// @Success 200 {html} html ""
// @Failure 500 {html} html "Внутренняя ошибка"
// @Router /ping [get]
func (h *MetricsHandler) Ping(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", ContentTypeTextHTML)

	if r.Method != http.MethodGet {
		logger.Log.Infow("error, Method != Get", "Method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	data, err := h.Service.Ping(r.Context())

	if err != nil {
		logger.Log.Infoln(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("<div>" + err.Error() + "</div>"))
		return
	}

	result := []byte("<div>")
	result = append(result, data...)
	result = append(result, []byte("</div>")...)
	w.Write(result)

}
