package usecase

import (
	"context"
	"errors"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserRepository interface {
	Create(ctx context.Context, name, email, passwordHash string, role entity.UserRole) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByID(ctx context.Context, id string) (*entity.User, error)
}

type RegisterInput struct {
	Name     string          `json:"name" binding:"required"`
	Email    string          `json:"email" binding:"required,email"`
	Password string          `json:"password" binding:"required,min=6"`
	Role     entity.UserRole `json:"role" binding:"required,oneof=admin customer"`
}

type RegisterOutput struct {
	User *entity.User `json:"user"`
}

type RegisterUserUseCase struct {
	repo UserRepository
}

func NewRegisterUserUseCase(repo UserRepository) *RegisterUserUseCase {
	return &RegisterUserUseCase{repo: repo}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	// Check if user already exists
	existing, err := uc.repo.GetByEmail(ctx, input.Email)
	if err == nil && existing != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Hash password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user, err := uc.repo.Create(ctx, input.Name, input.Email, string(hashedPassword), input.Role)
	if err != nil {
		return nil, err
	}

	return &RegisterOutput{User: user}, nil
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginOutput struct {
	User *entity.User `json:"user"`
}

type LoginUserUseCase struct {
	repo UserRepository
}

func NewLoginUserUseCase(repo UserRepository) *LoginUserUseCase {
	return &LoginUserUseCase{repo: repo}
}

func (uc *LoginUserUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// Find user by email
	user, err := uc.repo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Compare bcrypt password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return &LoginOutput{User: user}, nil
}
