package handlers

import (
	"net/http"

	"github.com/gabrielgcosta/ticketblast-core/internal/infra/web/apierror"
	"github.com/gabrielgcosta/ticketblast-core/internal/usecase"
	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	listActiveUC  *usecase.ListActiveEventsUseCase
	createEventUC *usecase.CreateEventUseCase
}

func NewEventHandler(
	listActiveUC *usecase.ListActiveEventsUseCase,
	createEventUC *usecase.CreateEventUseCase,
) *EventHandler {
	return &EventHandler{
		listActiveUC:  listActiveUC,
		createEventUC: createEventUC,
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

func (h *EventHandler) CreateEvent(c *gin.Context) {
	var input usecase.CreateEventInput
	if err := c.ShouldBindJSON(&input); err != nil {
		apierror.Write(c, apierror.BadRequest("Invalid request body", err))
		return
	}

	output, err := h.createEventUC.Execute(c.Request.Context(), input)
	if err != nil {
		apierror.Write(c, apierror.Internal("Failed to create event", err))
		return
	}

	c.JSON(http.StatusCreated, output)
}
