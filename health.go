package main

import (
	"net/http"

	"github.com/alexliesenfeld/health"
)

func HealthCheckHandler(repo Repository) http.HandlerFunc {
	checker := health.NewChecker()
	return health.NewHandler(checker)
}
