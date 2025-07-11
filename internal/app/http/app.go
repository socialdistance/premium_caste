package httpapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"premium_caste/internal/metrics"
	prommiddleware "premium_caste/internal/middleware"
	httprouters "premium_caste/internal/transport/http"
	"premium_caste/internal/transport/http/dto/response"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
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
	log          *slog.Logger
	e            *echo.Echo
	routers      *httprouters.Routers
	metricsReg   *prometheus.Registry
	metricServer *http.Server
	host         string
	port         string
	token        string
}

func New(log *slog.Logger, token string, host, port string, routers *httprouters.Routers) *Server {
	e := echo.New()
	e.HideBanner = true

	validate := validator.New()
	e.Validator = &CustomValidator{validator: validate}

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:4173"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
		AllowCredentials: true,
		MaxAge:           86400,
	}))
	// e.Use(session.Middleware(sessions.NewCookieStore([]byte("test"))))

	store := sessions.NewCookieStore([]byte("test"))
	store.Options = &sessions.Options{
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	}

	e.Use(session.Middleware(store))

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

	metricsServer := &http.Server{
		Addr:    ":9090",
		Handler: promhttp.HandlerFor(metricsReg, promhttp.HandlerOpts{}),
	}

	return &Server{
		log:          log,
		e:            e,
		routers:      routers,
		metricsReg:   metricsReg,
		metricServer: metricsServer,
		host:         host,
		port:         port,
		token:        token,
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
			s.log.Error("HTTP server stopped", op, slog.Any("error", err))
		}
	}()

	go func() {
		if err := s.metricServer.ListenAndServe(); err != nil {
			s.log.Error("HTTP server stopped", op, slog.Any("error", err))
		}
	}()

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

	if err := s.metricServer.Shutdown(optCtx); err != nil {
		return fmt.Errorf("metrics server shutdown failed: %w", err)
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

		sess.Options.MaxAge = 86400 * 7 // Обновляем срок
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			return c.JSON(http.StatusInternalServerError, "failed to save session")
		}

		return next(c)
	}
}

func (s *Server) jwtFromCookieMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("access_token")
		if err != nil {
			return c.JSON(http.StatusUnauthorized, response.ErrorResponse{
				Error: "access token required in cookies",
			})
		}

		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(s.token), nil
		})

		if err != nil || !token.Valid {
			return c.JSON(http.StatusUnauthorized, response.ErrorResponse{
				Error: "invalid or expired token",
			})
		}

		if _, ok := token.Claims.(jwt.MapClaims); ok {
			c.SetCookie(&http.Cookie{
				Name:     "access_token",
				Value:    cookie.Value,
				Expires:  time.Now().Add(1 * time.Hour),
				Path:     "/",
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteLaxMode,
			})
		}

		sess, _ := session.Get("session", c)
		sess.Options.MaxAge = 86400 * 7 // Обновляем срок
		sess.Save(c.Request(), c.Response())

		c.Set("user", token.Claims)

		return next(c)
	}
}

func (s *Server) BuildRouters() {
	s.e.GET("/metrics", echo.WrapHandler(promhttp.HandlerFor(s.metricsReg, promhttp.HandlerOpts{})))
	s.e.GET("/swagger/*", echoSwagger.WrapHandler)

	s.e.GET("/uploads/*", func(c echo.Context) error {
		filePath := path.Join("uploads", c.Param("*"))

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "File not found")
		}

		if strings.Contains(filePath, "../") {
			return echo.NewHTTPError(http.StatusForbidden, "Invalid path")
		}

		return c.File(filePath)
	})

	// s.e.Static("/uploads", "./uploads")
	// s.e.GET("/uploads/*", func(c echo.Context) error {
	// 	return s.adminOnlyMiddleware(func(c echo.Context) error {
	// 		filepath := "uploads/" + c.Param("*")
	// 		return c.File(filepath)
	// 	})(c)
	// }, s.jwtFromCookieMiddleware)

	api := s.e.Group("/api/v1")
	api.Use(prommiddleware.PrometheusMetrics)
	{
		api.POST("/register", s.routers.Register)
		api.POST("/login", s.routers.Login)
		api.POST("/refresh", s.routers.Refresh)

		userGroup := api.Group("/users")
		userGroup.Use(s.jwtFromCookieMiddleware)
		{
			userGroup.GET("/:user_id/is-admin", s.routers.IsAdminPermission)
			userGroup.POST("/user_id", s.routers.GetUserById)
		}

		mediaGroup := api.Group("/media", s.adminOnlyMiddleware)
		mediaGroup.Use(s.jwtFromCookieMiddleware)
		{
			mediaGroup.POST("/upload", s.routers.UploadMedia)
			mediaGroup.POST("/uploads", s.routers.UploadMultipleMedia)
			mediaGroup.POST("/groups/attach", s.routers.AttachMediaToGroup)
			mediaGroup.POST("/groups", s.routers.CreateMediaGroup)
			mediaGroup.GET("/groups/group_id", s.routers.ListGroupMedia)
			mediaGroup.GET("/images", s.routers.GetAllImages)
		}

		blogGroup := api.Group("/posts")
		blogGroup.GET("", s.routers.ListPosts)
		blogGroup.GET("/:id", s.routers.GetPost)
		blogGroup.GET("/:id/media-groups", s.routers.GetPostMediaGroups)
		blogGroup.Use(s.jwtFromCookieMiddleware)
		{
			blogGroup.POST("", s.routers.CreatePost, s.adminOnlyMiddleware)
			blogGroup.PUT("/:id", s.routers.UpdatePost, s.adminOnlyMiddleware)
			blogGroup.DELETE("/:id", s.routers.DeletePost, s.adminOnlyMiddleware)
			blogGroup.PATCH("/:id/publish", s.routers.PublishPost, s.adminOnlyMiddleware)
			blogGroup.PATCH("/:id/archive", s.routers.ArchivePost, s.adminOnlyMiddleware)
			blogGroup.POST("/:id/media-groups", s.routers.AddMediaGroup, s.adminOnlyMiddleware)
		}

		galleryGroup := api.Group("/gallery")
		galleryGroup.GET("/galleries", s.routers.GetGalleriesHandler)
		galleryGroup.GET("/galleries/:id", s.routers.GetGalleryByIDHandler)
		galleryGroup.GET("/galleries/by-tags", s.routers.GetGalleriesByTagsHandler)
		galleryGroup.Use(s.jwtFromCookieMiddleware)
		{
			galleryGroup.POST("/galleries", s.routers.CreateGalleryHandler, s.adminOnlyMiddleware)
			galleryGroup.PUT("/galleries", s.routers.UpdateGalleryHandler, s.adminOnlyMiddleware)
			galleryGroup.PATCH("/galleries/:id/status", s.routers.UpdateGalleryStatusHandler, s.adminOnlyMiddleware)
			galleryGroup.DELETE("/galleries/:id", s.routers.DeleteGalleryHandler, s.adminOnlyMiddleware)
		}
	}
}
