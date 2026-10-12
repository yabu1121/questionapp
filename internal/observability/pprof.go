package observability

import (
	"log/slog"
	"net/http"

	_ "net/http/pprof"
)

func StartPprof(logger *slog.Logger, address string) {
	logger.Info("pprof server listening", "address", address)

	if err := http.ListenAndServe(address, nil); err != nil {
		logger.Error("pprof server stopped", "error", err)
	}
}
