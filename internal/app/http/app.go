package httpapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httprouters "premium_caste/internal/transport/http"

	"github.com/arl/statsviz"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	echojwt "github.com/labstack/echo-jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	m       *http.ServeMux
	log     *slog.Logger
	e       *echo.Echo
	routers *httprouters.Routers
	host    string
	port    string
}

func New(log *slog.Logger, token string, host, port string, routers *httprouters.Routers) *Server {
	e := echo.New()
	e.HideBanner = true

	// e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
	//     AllowOrigins: []string{"*"},
	//     AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
	// }))

	e.Use(middleware.CORS())
	e.Use(middleware.Recover())

	// e.Use(echojwt.WithConfig(echojwt.Config{
	// SigningKey: []byte(token),
	// }))

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:      true,
		LogStatus:   true,
		LogRemoteIP: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info("request",
				slog.String("URI", v.URI),
				slog.Int("status", v.Status),
				slog.String("remote ip", v.RemoteIP),
			)

			return nil
		},
	}))

	mux := http.NewServeMux()
	err := statsviz.Register(mux)
	if err != nil {
		log.Info("Statsviz start with error", slog.Any("error:", err.Error()))
	}

	return &Server{
		m:       mux,
		log:     log,
		e:       e,
		routers: routers,
		host:    host,
		port:    port,
	}
}

// MustRun runs HTTP server and panics if any error occurs.
func (s *Server) MustRun() {
	const op = "http.Server.MustRun"

	s.log.Info(op, slog.String("Start", "server"))

	if err := s.Start(); err != nil {
		panic(err)
	}
}

func (s *Server) Start() error {
	const op = "http.Server.Start"

	if err := s.e.Start(fmt.Sprintf(":%s", s.port)); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("%s server stopped: %w", op, err)
	}

	return nil
}

func (s *Server) Stop() error {
	const op = "http.Server.Stop"

	optCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	s.log.Info("stopping", op, "http server")

	if err := s.e.Shutdown(optCtx); err != nil {
		return fmt.Errorf("%s could not shutdown server gracefuly: %w", op, err)
	}

	return nil
}

func (s *Server) jwtMiddleware() echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte("your-secret-key"), // Замените на реальный ключ из конфига
		TokenLookup: "header:Authorization",
	})
}

func (s *Server) adminOnlyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := jwt.New(jwt.SigningMethodHS256)

		claims := token.Claims.(jwt.MapClaims)

		userID, err := uuid.Parse(claims["sub"].(string))
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		isAdmin, err := s.routers.UserService.IsAdmin(c.Request().Context(), userID)
		if err != nil || !isAdmin {
			return echo.NewHTTPError(http.StatusForbidden, "admin access required")
		}

		return next(c)
	}
}

func (s *Server) BuildRouters() {
	// Debug endpoints
	debug := s.e.Group("/debug")
	debug.GET("/statsviz/", echo.WrapHandler(s.m))
	debug.GET("/statsviz/*", echo.WrapHandler(s.m))

	// API v1 routes
	api := s.e.Group("/api/v1")
	{
		userGroup := api.Group("/users")
		{
			// userGroup.POST("", s.routers.CreateUser)
			userGroup.POST("/login", s.routers.Login)
			// userGroup.GET("/:email", s.routers.GetUserByEmail, s.adminOnlyMiddleware)
		}

		// mediaGroup := api.Group("/media", s.jwtMiddleware())
		// {
		// 	mediaGroup.POST("", s.routers.CreateMedia)
		// 	mediaGroup.PUT("/:id", s.routers.UpdateMedia)
		// 	mediaGroup.GET("/:id", s.routers.GetMediaByID)
		// 	mediaGroup.POST("/groups", s.routers.CreateMediaGroup)
		// 	mediaGroup.POST("/groups/:groupID/items", s.routers.AddMediaToGroup)
		// 	mediaGroup.GET("/groups/:groupID", s.routers.GetMediaByGroupID)
		// }
	}
}
