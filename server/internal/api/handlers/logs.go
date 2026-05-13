package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nemvince/fog-next/ent"
	entagentlog "github.com/nemvince/fog-next/ent/agentlog"
	"github.com/nemvince/fog-next/internal/api/response"
)

// Logs handles log-related read endpoints available to authenticated users.
type Logs struct{ db *ent.Client }

func NewLogs(db *ent.Client) *Logs { return &Logs{db} }

// GetTaskLogs returns the agent log entries for a task, paginated by creation
// time.  Query params: limit (default 200, max 500), before (RFC3339Nano
// timestamp for cursor-based pagination).
func (h *Logs) GetTaskLogs(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "invalid task id")
		return
	}

	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 500 {
				n = 500
			}
			limit = n
		}
	}

	q := h.db.AgentLog.Query().
		Where(entagentlog.TaskIDEQ(taskID)).
		Order(entagentlog.ByLoggedAt()).
		Limit(limit)

	if before := r.URL.Query().Get("before"); before != "" {
		if t, err := time.Parse(time.RFC3339Nano, before); err == nil {
			q = q.Where(entagentlog.LoggedAtLT(t))
		}
	}

	logs, err := q.All(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	response.OK(w, response.ListOf(logs))
}
