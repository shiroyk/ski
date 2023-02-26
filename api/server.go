package api

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	v1 "github.com/shiroyk/cloudcat/api/v1"
	"golang.org/x/exp/slog"
)

const (
	// DefaultTimeout the default timeout
	DefaultTimeout = time.Minute
	// DefaultAddress the api default address
	DefaultAddress = "localhost:8080"
)

// Options the api server configuration
type Options struct {
	Logger  *slog.Logger  `yaml:"-"`
	Token   string        `yaml:"token"`
	Address string        `yaml:"address"`
	Timeout time.Duration `yaml:"timeout"`
}

// Server the api service
func Server(opt Options) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = errorHandler
	e.HideBanner = true
	e.Use(loggerMiddleware(opt), authMiddleware(opt))
	e.Any("/ping", ping)
	e.Any("", ping)
	v1.RouteAnalyze(e)
	return e
}

func errorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	}
	c.Logger().Error(err)

	if err = c.JSON(code, map[string]string{"msg": err.Error()}); err != nil {
		c.Logger().Error(err)
	}
}

func ping(ctx echo.Context) error {
	return ctx.NoContent(http.StatusOK)
}
