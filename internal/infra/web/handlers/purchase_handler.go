package handlers

import (
	"errors"
	"net/http"

	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/apierror"
	"github.com/gabrielgcosta/ticketblast-core/internal/usecase"
	"github.com/gin-gonic/gin"
)

type PurchaseHandler struct {
	purchaseUC *usecase.PurchaseUseCase
}

func NewPurchaseHandler(
	purchaseUC *usecase.PurchaseUseCase,
) *PurchaseHandler {
	return &PurchaseHandler{
		purchaseUC: purchaseUC,
	}
}

func (h *PurchaseHandler) Purchase(c *gin.Context) {
	var userIDStr string
	if userIDVal, exists := c.Get("user_id"); exists {
		if id, ok := userIDVal.(string); ok {
			userIDStr = id
		}
	}
	if userIDStr == "" {
		apierror.Write(c, apierror.Unauthorized("User not authenticated", nil))
		return
	}

	var req struct {
		EventID  string `json:"event_id" binding:"required"`
		TicketID string `json:"ticket_id" binding:"required"`
		Quantity int    `json:"quantity" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.Write(c, apierror.BadRequest("Invalid request body", err))
		return
	}

	input := usecase.PurchaseInput{
		UserID:   userIDStr,
		EventID:  req.EventID,
		TicketID: req.TicketID,
		Quantity: req.Quantity,
	}

	output, err := h.purchaseUC.Execute(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, usecase.ErrSoldOut) {
			apierror.Write(c, apierror.UnprocessableEntity("Tickets sold out", err))
			return
		}
		if errors.Is(err, usecase.ErrTicketNotFound) || errors.Is(err, usecase.ErrTicketEventMismatch) || errors.Is(err, usecase.ErrInvalidQuantity) {
			apierror.Write(c, apierror.BadRequest("Invalid purchase parameters", err))
			return
		}
		apierror.Write(c, apierror.Internal("Failed to purchase tickets", err))
		return
	}

	c.JSON(http.StatusCreated, output)
}
