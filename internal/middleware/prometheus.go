package middleware

import (
	"premium_caste/internal/metrics"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func PrometheusMetrics(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		duration := time.Since(start).Seconds()

		metrics.HTTPRequestsTotal.WithLabelValues(
			c.Request().Method,
			c.Path(),
			strconv.Itoa(c.Response().Status),
		).Inc()

		metrics.HTTPRequestDuration.WithLabelValues(
			c.Request().Method,
			c.Path(),
		).Observe(duration)

		return err
	}
}
