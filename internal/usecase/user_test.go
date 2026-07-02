package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
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

func TestRegisterUserUseCase_Success(t *testing.T) {
	mockRepo := &mockUserRepository{
		getByEmailFn: func(ctx context.Context, email string) (*entity.User, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, name, email, passwordHash string, role entity.UserRole) (*entity.User, error) {
			assert.Equal(t, "Alice", name)
			assert.Equal(t, "alice@example.com", email)
			assert.Equal(t, entity.RoleCustomer, role)

			err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("password123"))
			assert.NoError(t, err)

			return &entity.User{
				ID:        "user-id",
				Name:      name,
				Email:     email,
				Role:      role,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	uc := NewRegisterUserUseCase(mockRepo)
	output, err := uc.Execute(context.Background(), RegisterInput{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
		Role:     entity.RoleCustomer,
	})

	assert.NoError(t, err)
	assert.NotNil(t, output.User)
	assert.Equal(t, "user-id", output.User.ID)
	assert.Equal(t, "Alice", output.User.Name)
}

func TestRegisterUserUseCase_EmailExists(t *testing.T) {
	mockRepo := &mockUserRepository{
		getByEmailFn: func(ctx context.Context, email string) (*entity.User, error) {
			return &entity.User{ID: "existing-id", Email: email}, nil
		},
		createFn: func(ctx context.Context, name, email, passwordHash string, role entity.UserRole) (*entity.User, error) {
			t.Fatal("Create should not be called")
			return nil, nil
		},
	}

	uc := NewRegisterUserUseCase(mockRepo)
	output, err := uc.Execute(context.Background(), RegisterInput{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "password123",
		Role:     entity.RoleCustomer,
	})

	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
	assert.Nil(t, output)
}

func TestLoginUserUseCase_Success(t *testing.T) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	mockRepo := &mockUserRepository{
		getByEmailFn: func(ctx context.Context, email string) (*entity.User, error) {
			return &entity.User{
				ID:           "user-id",
				Email:        email,
				PasswordHash: string(hashedPassword),
				Role:         entity.RoleCustomer,
			}, nil
		},
	}

	uc := NewLoginUserUseCase(mockRepo)
	output, err := uc.Execute(context.Background(), LoginInput{
		Email:    "alice@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, output.User)
	assert.Equal(t, "user-id", output.User.ID)
}

func TestLoginUserUseCase_InvalidCredentials(t *testing.T) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	mockRepo := &mockUserRepository{
		getByEmailFn: func(ctx context.Context, email string) (*entity.User, error) {
			return &entity.User{
				ID:           "user-id",
				Email:        email,
				PasswordHash: string(hashedPassword),
				Role:         entity.RoleCustomer,
			}, nil
		},
	}

	uc := NewLoginUserUseCase(mockRepo)

	// Wrong password
	output, err := uc.Execute(context.Background(), LoginInput{
		Email:    "alice@example.com",
		Password: "wrongpassword",
	})
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, output)

	// Non-existent email
	mockRepoNoUser := &mockUserRepository{
		getByEmailFn: func(ctx context.Context, email string) (*entity.User, error) {
			return nil, ErrInvalidCredentials
		},
	}
	ucNoUser := NewLoginUserUseCase(mockRepoNoUser)
	output, err = ucNoUser.Execute(context.Background(), LoginInput{
		Email:    "nonexistent@example.com",
		Password: "password123",
	})
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Nil(t, output)
}
