package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/spiderocious/chat-to-order/backend/internal/ordering"
	"github.com/spiderocious/chat-to-order/backend/internal/store"
)

// Error envelope: three jobs, three fields. code is the contract, message is
// what a person reads, severity is for dashboards.
type apiError struct {
	Code        string       `json:"code"`
	Message     string       `json:"message"`
	Severity    string       `json:"severity"`
	FieldErrors []fieldError `json:"field_errors,omitempty"`
	RequestID   string       `json:"request_id,omitempty"`
	Current     any          `json:"current,omitempty"`
	RetryAfter  int          `json:"retry_after,omitempty"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func ok(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

func okPage(c *gin.Context, data any, next string, more bool) {
	var nc *string
	if next != "" {
		nc = &next
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "meta": gin.H{"next_cursor": nc, "has_more": more}})
}

func fail(c *gin.Context, status int, code, msg, severity string) {
	c.AbortWithStatusJSON(status, gin.H{"error": apiError{Code: code, Message: msg, Severity: severity, RequestID: c.GetString("request_id")}})
}

func failWith(c *gin.Context, status int, e apiError) {
	e.RequestID = c.GetString("request_id")
	c.AbortWithStatusJSON(status, gin.H{"error": e})
}

func validation(c *gin.Context, errs ...fieldError) {
	msg := "Some fields are invalid."
	if len(errs) == 1 {
		msg = errs[0].Field + ": " + errs[0].Message
	}
	failWith(c, http.StatusBadRequest, apiError{Code: "validation", Message: msg, Severity: "body-validation", FieldErrors: errs})
}

// renderErr maps domain and store errors to the envelope.
func renderErr(c *gin.Context, err error) {
	var de *ordering.DomainError
	switch {
	case errors.As(err, &de):
		status, sev := http.StatusUnprocessableEntity, "business-rule"
		switch de.Code {
		case "validation":
			status, sev = http.StatusBadRequest, "body-validation"
		case "not_found", "unknown_reference":
			status, sev = http.StatusNotFound, "not-found"
		case "invalid_state_transition":
			status, sev = http.StatusConflict, "conflict"
		case "paystack_unavailable", "store_unavailable":
			status, sev = http.StatusBadGateway, "upstream"
		}
		fail(c, status, de.Code, de.Message, sev)
	case errors.Is(err, store.ErrNotFound):
		fail(c, http.StatusNotFound, "not_found", "Not found.", "not-found")
	case errors.Is(err, store.ErrConflict):
		fail(c, http.StatusConflict, "conflict", "A record with that id already exists.", "conflict")
	case errors.Is(err, store.ErrVersionConflict):
		fail(c, http.StatusConflict, "version_conflict", "Changed elsewhere. Reload and try again.", "conflict")
	default:
		c.Error(err) //nolint:errcheck
		fail(c, http.StatusInternalServerError, "internal", "Something went wrong on our side. Try again.", "server-fault")
	}
}
