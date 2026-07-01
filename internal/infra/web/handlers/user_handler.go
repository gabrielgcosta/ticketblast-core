package handlers

import (
	"errors"
	"net/http"

	"github.com/gabrielgcosta/ticketblast-core/internal/infra/auth"
	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/apierror"
	"github.com/gabrielgcosta/ticketblast-core/internal/usecase"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	registerUC  *usecase.RegisterUserUseCase
	loginUC     *usecase.LoginUserUseCase
	tokenEngine *auth.TokenEngine
}

func NewUserHandler(
	registerUC *usecase.RegisterUserUseCase,
	loginUC *usecase.LoginUserUseCase,
	tokenEngine *auth.TokenEngine,
) *UserHandler {
	return &UserHandler{
		registerUC:  registerUC,
		loginUC:     loginUC,
		tokenEngine: tokenEngine,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var input usecase.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Write(c, apierror.BadRequest("Invalid request body", err))
		return
	}

	output, err := h.registerUC.Execute(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, usecase.ErrEmailAlreadyExists) {
			apierror.Write(c, apierror.Conflict("Email is already registered", err))
			return
		}
		apierror.Write(c, apierror.Internal("Failed to register user", err))
		return
	}

	c.JSON(http.StatusCreated, output)
}

func (h *UserHandler) Login(c *gin.Context) {
	var input usecase.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Write(c, apierror.BadRequest("Invalid request body", err))
		return
	}

	output, err := h.loginUC.Execute(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			apierror.Write(c, apierror.Unauthorized("Invalid email or password", err))
			return
		}
		apierror.Write(c, apierror.Internal("Failed to authenticate user", err))
		return
	}

	// Generate JWT token
	token, err := h.tokenEngine.GenerateToken(output.User.ID, string(output.User.Role))
	if err != nil {
		apierror.Write(c, apierror.Internal("Failed to generate access token", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  output.User,
	})
}
