package handlers

import (
	"net/http"

	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/apierror"
	"github.com/gabrielgcosta/ticketblast-core/internal/usecase"
	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	listActiveUC *usecase.ListActiveEventsUseCase
}

func NewEventHandler(listActiveUC *usecase.ListActiveEventsUseCase) *EventHandler {
	return &EventHandler{
		listActiveUC: listActiveUC,
	}
}

func (h *EventHandler) ListActive(c *gin.Context) {
	output, err := h.listActiveUC.Execute(c.Request.Context())
	if err != nil {
		apierror.Write(c, apierror.Internal("Failed to list active events", err))
		return
	}

	c.JSON(http.StatusOK, output)
}
