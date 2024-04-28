package main

import (
	"net/http"

	"github.com/alexliesenfeld/health"
)

func HealthCheckHandler() http.HandlerFunc {
	checker := health.NewChecker()
	return health.NewHandler(checker)
}
