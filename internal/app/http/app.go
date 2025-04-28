package httpapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	httprouters "premium_caste/internal/transport/http"

	"github.com/arl/statsviz"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echojwt "github.com/labstack/echo-jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

type Server struct {
	m       *http.ServeMux
	log     *slog.Logger
	e       *echo.Echo
	routers *httprouters.Routers
	host    string
	port    string
	token   string
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
		token:   token,
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

func (s *Server) adminOnlyMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("session", c)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "session required"})
		}

		userID, ok := sess.Values["user_id"].(string)
		if !ok || userID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		parsedUUID, err := uuid.Parse(userID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID format"})
		}

		isAdmin, err := s.routers.UserService.IsAdmin(c.Request().Context(), parsedUUID)
		if err != nil || !isAdmin {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "admin access required"})
		}

		return next(c)
	}
}

func (s *Server) BuildRouters() {
	api := s.e.Group("/api/v1")
	{
		api.POST("/register", s.routers.Register)
		api.POST("/login", s.routers.Login)
		api.POST("/refresh", s.routers.Refresh)

		debug := s.e.Group("/debug")
		{
			debug.GET("/statsviz/", echo.WrapHandler(s.m))
			debug.GET("/statsviz/*", echo.WrapHandler(s.m))
		}

		swagger := s.e.Group("/swag")
		{
			swagger.GET("/swagger/*", echoSwagger.WrapHandler)
		}

		userGroup := api.Group("/users")
		userGroup.Use(echojwt.WithConfig(echojwt.Config{
			SigningKey: []byte(s.token),
		}))
		{
			userGroup.GET("/:user_id/is-admin", s.routers.IsAdminPermission)
			// userGroup.GET("/:email", s.routers.GetUserByEmail, s.adminOnlyMiddleware)
		}

		mediaGroup := api.Group("/media", s.adminOnlyMiddleware)
		{
			mediaGroup.POST("/upload", s.routers.UploadMedia)
			mediaGroup.POST("/groups/attach", s.routers.AttachMediaToGroup)
			mediaGroup.POST("/groups", s.routers.CreateMediaGroup)
			mediaGroup.GET("/groups/group_id", s.routers.ListGroupMedia)
		}
	}
}
