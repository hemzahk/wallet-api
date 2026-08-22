package health

import (
	"net/http"

	"github.com/hemzahk/wallet-api/internal/json"
)

type handler struct {
	config Config
}

type Config struct {
	Env string
	Version string
}

func NewHandler(config Config) *handler {
	return &handler{config : config}
}

func (h *handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status":  "ok",
		"env":     h.config.Env,
		"version": h.config.Version,
	}

	if err := json.JsonResponse(w, http.StatusOK, data); err != nil {
		json.InternalServerError(w, r, err)
	}
}