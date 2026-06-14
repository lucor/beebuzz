package server

import (
	"net/http"

	"go.beebuzz.app/beebuzz/docs"
)

const openAPIYAMLContentType = "application/yaml; charset=utf-8"

func openAPIYAML(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", openAPIYAMLContentType)
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(docs.OpenAPIYAML)
}
