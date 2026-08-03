package handler

import (
	"net/http"

	"github.com/devjaime/control-propiedades/apps/api/cloudhandler"
)

// Handler is the Vercel Function entrypoint.
func Handler(w http.ResponseWriter, r *http.Request) {
	cloudhandler.Handler(w, r)
}
