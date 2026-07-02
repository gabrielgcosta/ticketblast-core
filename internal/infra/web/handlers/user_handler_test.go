package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"github.com/gabrielgcosta/ticketblast-core/internal/infra/auth"
	"github.com/gabrielgcosta/ticketblast-core/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepository struct {
	createFn     func(ctx context.Context, name, email, passwordHash string, role entity.UserRole) (*entity.User, error)
	getByEmailFn func(ctx context.Context, email string) (*entity.User, error)
	getByIDFn    func(ctx context.Context, id string) (*entity.User, error)
}

func (m *mockUserRepository) Create(ctx context.Context, name, email, passwordHash string, role entity.UserRole) (*entity.User, error) {
	return m.createFn(ctx, name, email, passwordHash, role)
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	return m.getByEmailFn(ctx, email)
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	return m.getByIDFn(ctx, id)
}

func TestUserHandler_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := &mockUserRepository{
		getByEmailFn: func(ctx context.Context, email string) (*entity.User, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, name, email, passwordHash string, role entity.UserRole) (*entity.User, error) {
			return &entity.User{
				ID:    "user-123",
				Name:  name,
				Email: email,
				Role:  role,
			}, nil
		},
	}

	registerUC := usecase.NewRegisterUserUseCase(mockRepo)
	loginUC := usecase.NewLoginUserUseCase(mockRepo)
	tokenEngine := auth.NewTokenEngine("secret")
	handler := NewUserHandler(registerUC, loginUC, tokenEngine)

	r := gin.New()
	r.POST("/register", handler.Register)

	input := usecase.RegisterInput{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
		Role:     entity.RoleCustomer,
	}
	body, _ := json.Marshal(input)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"user-123"`)
	assert.Contains(t, w.Body.String(), `"email":"alice@example.com"`)
}

func TestUserHandler_Register_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenEngine := auth.NewTokenEngine("secret")
	handler := NewUserHandler(nil, nil, tokenEngine)

	r := gin.New()
	r.POST("/register", handler.Register)

	body := []byte(`{"name":""}`)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestUserHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	mockRepo := &mockUserRepository{
		getByEmailFn: func(ctx context.Context, email string) (*entity.User, error) {
			return &entity.User{
				ID:           "user-123",
				Name:         "Alice",
				Email:        email,
				PasswordHash: string(hashedPassword),
				Role:         entity.RoleCustomer,
			}, nil
		},
	}

	registerUC := usecase.NewRegisterUserUseCase(mockRepo)
	loginUC := usecase.NewLoginUserUseCase(mockRepo)
	tokenEngine := auth.NewTokenEngine("secret")
	handler := NewUserHandler(registerUC, loginUC, tokenEngine)

	r := gin.New()
	r.POST("/login", handler.Login)

	input := usecase.LoginInput{
		Email:    "alice@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(input)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"token":`)
	assert.Contains(t, w.Body.String(), `"email":"alice@example.com"`)
}
