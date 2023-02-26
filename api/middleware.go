package api

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/exp/slog"
)

func loggerMiddleware(opt Options) echo.MiddlewareFunc {
	log := opt.Logger
	if log == nil {
		log = slog.Default()
	}
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true,
		LogURI:    true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info("request", "uri", v.URI, "status", v.Status)
			return nil
		},
	})
}

func authMiddleware(opt Options) echo.MiddlewareFunc {
	return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup:  "header:" + echo.HeaderAuthorization,
		AuthScheme: "Bearer",
		Validator: func(auth string, c echo.Context) (bool, error) {
			return opt.Token == "" || auth == opt.Token, nil
		},
	})
}
