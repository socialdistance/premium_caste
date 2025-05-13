package httpapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"premium_caste/internal/metrics"
	prommiddleware "premium_caste/internal/middleware"
	httprouters "premium_caste/internal/transport/http"
	"premium_caste/internal/transport/http/dto/response"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echojwt "github.com/labstack/echo-jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

type Server struct {
	log        *slog.Logger
	e          *echo.Echo
	routers    *httprouters.Routers
	metricsReg *prometheus.Registry
	host       string
	port       string
	token      string
}

func New(log *slog.Logger, token string, host, port string, routers *httprouters.Routers) *Server {
	e := echo.New()
	e.HideBanner = true

	validate := validator.New()
	e.Validator = &CustomValidator{validator: validate}

	// e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
	//     AllowOrigins: []string{"*"},
	//     AllowMethods: []string{echo.GET, echo.PUT, echo.POST, echo.DELETE},
	// }))

	e.Use(session.Middleware(sessions.NewCookieStore([]byte("test"))))

	e.Use(middleware.CORS())
	e.Use(middleware.Recover())

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

	metricsReg := prometheus.NewRegistry()
	metrics.RegisterMetrics(metricsReg)

	return &Server{
		log:        log,
		e:          e,
		routers:    routers,
		metricsReg: metricsReg,
		host:       host,
		port:       port,
		token:      token,
	}
}

func (s *Server) MustRun() {
	const op = "http.Server.MustRun"

	s.log.Info(op, slog.String("Start", "server"))

	if err := s.Start(); err != nil {
		panic(err)
	}
}

func (s *Server) Start() error {
	const op = "http.Server.Start"

	go func() {
		if err := s.e.Start(fmt.Sprintf(":%s", s.port)); err != nil && err != http.ErrServerClosed {
			s.log.Error("%s HTTP server stopped", op, slog.Any("error", err))
		}
	}()

	// Запуск метрик сервера
	metricsServer := &http.Server{
		Addr:    ":9090",
		Handler: promhttp.HandlerFor(s.metricsReg, promhttp.HandlerOpts{}),
	}

	go func() {
		if err := metricsServer.ListenAndServe(); err != nil {
			s.log.Error("%s HTTP server stopped", op, slog.Any("error", err))
		}
	}()

	return nil

	// if err := s.e.Start(fmt.Sprintf(":%s", s.port)); err != nil && err != http.ErrServerClosed {
	// 	return fmt.Errorf("%s server stopped: %w", op, err)
	// }

	// return nil
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

func (s *Server) adminOnlyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("session", c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, response.ErrorResponse{Error: "session required"})
		}

		userID, ok := sess.Values["user_id"].(string)
		if !ok || userID == "" {
			return c.JSON(http.StatusUnauthorized, response.ErrorResponse{Error: "authentication required"})
		}

		parsedUUID, err := uuid.Parse(userID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, response.ErrorResponse{Error: "invalid user ID format"})
		}

		isAdmin, err := s.routers.UserService.IsAdmin(c.Request().Context(), parsedUUID)
		if err != nil || !isAdmin {
			return c.JSON(http.StatusForbidden, response.ErrorResponse{Error: "admin access required"})
		}

		return next(c)
	}
}

func (s *Server) BuildRouters() {
	s.e.GET("/metrics", echo.WrapHandler(promhttp.HandlerFor(s.metricsReg, promhttp.HandlerOpts{})))
	s.e.GET("/swagger/*", echoSwagger.WrapHandler)

	api := s.e.Group("/api/v1")
	api.Use(prommiddleware.PrometheusMetrics)
	{
		api.POST("/register", s.routers.Register)
		api.POST("/login", s.routers.Login)
		api.POST("/refresh", s.routers.Refresh)

		userGroup := api.Group("/users")
		userGroup.Use(echojwt.WithConfig(echojwt.Config{
			SigningKey: []byte(s.token),
		}))
		{
			userGroup.GET("/:user_id/is-admin", s.routers.IsAdminPermission)
			// userGroup.GET("/:email", s.routers.GetUserByEmail, s.adminOnlyMiddleware)
		}

		mediaGroup := api.Group("/media", s.adminOnlyMiddleware)
		mediaGroup.Use(echojwt.WithConfig(echojwt.Config{
			SigningKey: []byte(s.token),
		}))
		{
			mediaGroup.POST("/upload", s.routers.UploadMedia)
			mediaGroup.POST("/groups/attach", s.routers.AttachMediaToGroup)
			mediaGroup.POST("/groups", s.routers.CreateMediaGroup)
			mediaGroup.GET("/groups/group_id", s.routers.ListGroupMedia)
		}

		blogGroup := api.Group("/posts")
		blogGroup.GET("", s.routers.ListPosts)
		blogGroup.Use(echojwt.WithConfig(echojwt.Config{
			SigningKey: []byte(s.token),
		}))
		{
			blogGroup.POST("", s.routers.CreatePost, s.adminOnlyMiddleware)
			blogGroup.GET("/:id", s.routers.GetPost)
			blogGroup.PUT("/:id", s.routers.UpdatePost, s.adminOnlyMiddleware)
			blogGroup.DELETE("/:id", s.routers.DeletePost, s.adminOnlyMiddleware)
			blogGroup.PATCH("/:id/publish", s.routers.PublishPost, s.adminOnlyMiddleware)
			blogGroup.PATCH("/:id/archive", s.routers.ArchivePost, s.adminOnlyMiddleware)
			blogGroup.POST("/:id/media-groups", s.routers.AddMediaGroup, s.adminOnlyMiddleware)
			blogGroup.GET("/:id/media-groups", s.routers.GetPostMediaGroups, s.adminOnlyMiddleware)
		}
	}
}
