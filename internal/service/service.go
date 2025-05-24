package service

import (
	model2 "GoMLServe/gen_go/proto/model"
	"GoMLServe/internal/domain/models"
	"GoMLServe/internal/repository"
	"GoMLServe/pkg/jwt"
	"GoMLServe/pkg/logger"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
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
	Predict(predict *models.Phrase, token string) (string, error)
	GetResult(token string, id string) (*models.Result, error)
	sendToRabbitMQ(id string, email string, text string) error
	processTask(body []byte)
	sendToMLServer(expression string) (*models.Result, error)
	StartMLTaskConsumer() error
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

func (s *MLService) Predict(predict *models.Phrase, token string) (string, error) {
	email, err := jwt.GetEmailFromToken(token)
	if err != nil {
		return "", err
	}
	logger.GetLoggerFromCtx(s.ctx).Info("predict")
	id := uuid.New().String()
	err = s.repository.SavePhrase(id, email, predict)
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error saving phrase: %v", zap.Error(err))
		return "", fmt.Errorf("error saving phrase: %v", err)
	}
	err = s.sendToRabbitMQ(id, email, predict.Text)
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error sending to rabbitmq: %v", zap.Error(err))
		return "", fmt.Errorf("error sending to rabbitmq: %v", err)
	}
	logger.GetLoggerFromCtx(s.ctx).Info("phrase saved")
	return id, nil
}

func (s *MLService) GetResult(token string, id string) (*models.Result, error) {
	email, err := jwt.GetEmailFromToken(token)
	if err != nil {
		return nil, err
	}
	logger.GetLoggerFromCtx(s.ctx).Info("get result")
	result, err := s.repository.GetResult(email, id)
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("error getting result: %v", zap.Error(err))
		return nil, fmt.Errorf("error getting result: %v", err)
	}
	if result == nil {
		return nil, fmt.Errorf("result not found")
	}
	logger.GetLoggerFromCtx(s.ctx).Info("result found")
	return result, nil
}

func (s *MLService) sendToRabbitMQ(id string, email string, text string) error {
	conn, err := amqp.Dial("amqp://admin:admin123@localhost:5672/")
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	q, err := ch.QueueDeclare(
		"ml_tasks",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	body := fmt.Sprintf(`{"id":"%s","email":"%s","expression":"%s"}`, id, email, text)
	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
		},
	)
	return err
}

func (s *MLService) StartMLTaskConsumer() error {
	url := "amqp://admin:admin123@localhost:5672/"

	config := amqp.Config{
		Dial: amqp.DefaultDial(time.Second * 5),
		Properties: amqp.Table{
			"connection_name": "ml-service-connection",
		},
	}

	var conn *amqp.Connection
	var err error

	for i := 0; i < 3; i++ {
		conn, err = amqp.DialConfig(url, config)
		if err == nil {
			logger.GetLoggerFromCtx(s.ctx).Info("Connected to RabbitMQ")
			break
		}
		logger.GetLoggerFromCtx(s.ctx).Error("Failed to connect to RabbitMQ 1 time", zap.Error(err))
		time.Sleep(time.Second * 2)
	}
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Fatal("Failed to connect to RabbitMQ", zap.Error(err))
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Fatal("Failed to open a channel", zap.Error(err))
		return err
	}
	defer ch.Close()

	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
			q, err := ch.QueueDeclare(
				"ml_tasks",
				true,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				logger.GetLoggerFromCtx(s.ctx).Fatal("Failed to declare a queue", zap.Error(err))
				return err
			}

			msgs, err := ch.Consume(
				q.Name,
				"",
				false,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				logger.GetLoggerFromCtx(s.ctx).Fatal("Failed to register a consumer", zap.Error(err))
				return err
			}

			for msg := range msgs {
				s.processTask(msg.Body)
				msg.Ack(false)
			}
			return nil
		}
	}
}

func (s *MLService) processTask(body []byte) {
	var task struct {
		ID         string `json:"id"`
		Email      string `json:"email"`
		Expression string `json:"expression"`
	}

	if err := json.Unmarshal(body, &task); err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("Failed to parse task", zap.Error(err))
		return
	}

	result, err := s.sendToMLServer(task.Expression)
	if err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("ML processing failed",
			zap.String("id", task.ID),
			zap.Error(err),
		)
		return
	}

	if err = s.repository.UpdateResult(task.ID, task.Email, result); err != nil {
		logger.GetLoggerFromCtx(s.ctx).Error("Failed to save result",
			zap.String("id", task.ID),
			zap.Error(err),
		)
	}
}

func (s *MLService) sendToMLServer(expression string) (*models.Result, error) {
	if expression == "" {
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
	logger.GetLoggerFromCtx(s.ctx).Info("predict", zap.String("text", expression))
	res, err := c.Predict(ctx, &model2.BertRequest{Text: expression})
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
	if res.Logits[0] > res.Logits[1] && res.Logits[0] > res.Logits[2] {
		return &models.Result{Text: expression, Res: "neutral"}, nil
	} else if res.Logits[1] > res.Logits[0] && res.Logits[1] > res.Logits[2] {
		return &models.Result{Text: expression, Res: "positive"}, nil
	}
	return &models.Result{Text: expression, Res: "negative"}, nil
}
