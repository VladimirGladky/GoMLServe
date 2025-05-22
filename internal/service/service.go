package service

import (
	model2 "GoMLServe/gen_go/proto/model"
	"GoMLServe/internal/domain/models"
	"GoMLServe/internal/repository"
	"GoMLServe/pkg/jwt"
	"GoMLServe/pkg/logger"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type MLServiceInterface interface {
	Login(login *models.Login) (string, error)
	Register(register *models.Register) error
	Predict(predict *models.Phrase) (*models.Predict, error)
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

func (s *MLService) Predict(predict *models.Phrase) (*models.Predict, error) {
	if predict.Text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error creating grpc client: %v", zap.Error(err))
		return nil, fmt.Errorf("error creating grpc client: %v", err)
	}
	defer conn.Close()
	c := model2.NewBertServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.GetLoggerFromCtx(s.ctx).Info("predict", zap.String("text", predict.Text))
	res, err := c.Predict(ctx, &model2.BertRequest{Text: predict.Text})
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error making grpc request: %v", zap.Error(err))
		return nil, fmt.Errorf("error making grpc request: %v", err)
	}
	if res == nil {
		return nil, fmt.Errorf("empty response from server")
	}
	if len(res.Logits) == 0 {
		return nil, fmt.Errorf("server returned no logits")
	}
	return &models.Predict{Res: res.Logits}, nil
}
