package service

import (
	"GoMLServe/internal/domain/models"
	"GoMLServe/internal/repository"
	"GoMLServe/pkg/jwt"
	"GoMLServe/pkg/logger"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type MLServiceInterface interface {
	Login(login *models.Login) (string, error)
	Register(register *models.Register) error
}

type MLService struct {
	ctx        context.Context
	repository repository.MLRepositoryInterface
}

func New(ctx context.Context, repository repository.MLRepositoryInterface) *MLService {
	return &MLService{
		ctx:        ctx,
		repository: repository,
	}
}

func (s *MLService) Login(login *models.Login) (string, error) {
	logger.GetLoggerFromCtx(s.ctx).Info("login")
	if err := login.Validate(); err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error validating login: %v", zap.Error(err))
		return "", err
	}
	user, err := s.repository.User(login.Username)
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error getting user: %v", zap.Error(err))
		return "", fmt.Errorf("error getting user: %v", err)
	}
	err = bcrypt.CompareHashAndPassword(user.PassHash, []byte(login.Password))
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("wrong login or password")
		return "", ErrInvalidCredentials
	}
	token, err := jwt.NewToken(user, 24*time.Hour)
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error creating token: %v", zap.Error(err))
		return "", fmt.Errorf("error creating token: %v", err)
	}
	logger.GetLoggerFromCtx(s.ctx).Info("user logged in")
	return token, nil
}

func (s *MLService) Register(register *models.Register) error {
	logger.GetLoggerFromCtx(s.ctx).Info("register")
	if err := register.Validate(); err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error validating register request: %v", zap.Error(err))
		return fmt.Errorf("error validating register request: %v", err)
	}
	passHash, err := bcrypt.GenerateFromPassword([]byte(register.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error hashing password: %v", zap.Error(err))
		return fmt.Errorf("error hashing password: %v", err)
	}
	err = s.repository.SaveUser(&models.User{
		Email:    register.Username,
		PassHash: passHash,
	})
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error saving user: %v", zap.Error(err))
		return fmt.Errorf("error saving user: %v", err)
	}
	logger.GetLoggerFromCtx(s.ctx).Info("user registered")
	return nil
}
