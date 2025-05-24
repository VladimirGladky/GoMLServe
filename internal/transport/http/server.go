package http

import (
	"GoMLServe/internal/config"
	"GoMLServe/internal/domain/models"
	"GoMLServe/internal/repository"
	"GoMLServe/internal/service"
	"GoMLServe/pkg/logger"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

type MLServer struct {
	MLService service.MLServiceInterface
	cfg       *config.Config
	ctx       context.Context
}

func New(cfg *config.Config, ctx context.Context, MLService service.MLServiceInterface) *MLServer {
	return &MLServer{
		MLService: MLService, cfg: cfg, ctx: ctx}
}

func (s *MLServer) Run() error {
	router := gin.Default()
	router.POST("/api/v1/register", s.RegisterHandler())
	router.POST("/api/v1/login", s.LoginHandler())
	router.POST("/api/v1/predict", s.PredictHandler())
	router.GET("/api/v1/result/:id", s.GetResultHandler())
	err := router.Run(s.cfg.Host + ":" + s.cfg.Port)
	if err != nil {
		return fmt.Errorf("unable to start server: %w", err)
	}
	return nil
}

func (s *MLServer) RegisterHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.JSON(500, gin.H{
					"error": rec,
				})
			}
		}()
		if c.Request.Method != "POST" {
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"error": "method not allowed",
			})
			return
		}
		var reg *models.Register
		if err := c.ShouldBindJSON(&reg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		err := s.MLService.Register(reg)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
		})
		return
	}
}

func (s *MLServer) LoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.JSON(500, gin.H{
					"error": rec,
				})
			}
		}()
		if c.Request.Method != "POST" {
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"error": "method not allowed",
			})
			return
		}
		var login *models.Login
		if err := c.ShouldBindJSON(&login); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		token, err := s.MLService.Login(login)
		if err != nil {
			if errors.Is(err, repository.ErrUserNotFound) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": err.Error(),
				})
				return
			}
			if errors.Is(err, service.ErrInvalidCredentials) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"token": token,
		})
	}
}

func (s *MLServer) PredictHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.JSON(500, gin.H{
					"error": rec,
				})
			}
		}()
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			return
		}
		tokenString := parts[1]
		if c.Request.Method != "POST" {
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"error": "method not allowed",
			})
			return
		}
		var text *models.Phrase
		if err := c.ShouldBindJSON(&text); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		id, err := s.MLService.Predict(text, tokenString)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"id": id,
		})
	}
}

func (s *MLServer) GetResultHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				c.JSON(500, gin.H{
					"error": rec,
				})
			}
		}()
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			return
		}
		tokenString := parts[1]
		if c.Request.Method != "GET" {
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"error": "method not allowed",
			})
			return
		}
		id := c.Param("id")
		logger.GetLoggerFromCtx(s.ctx).Info("get id", zap.String("id", id))
		result, err := s.MLService.GetResult(tokenString, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"result": result,
		})
	}
}
