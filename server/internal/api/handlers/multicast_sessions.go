package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/ckAdmins/fog-next/ent"
	enttask "github.com/ckAdmins/fog-next/ent/task"
	"github.com/ckAdmins/fog-next/internal/api/response"
)

// MulticastSessions handles CRUD for multicast sessions.
type MulticastSessions struct {
	db *ent.Client
}

func NewMulticastSessions(db *ent.Client) *MulticastSessions {
	return &MulticastSessions{db: db}
}

type createMulticastSessionRequest struct {
	Name      string      `json:"name"`
	ImageID   string      `json:"imageId"`
	HostIDs   []string    `json:"hostIds"`
	Portbase  int         `json:"portbase,omitempty"`
}

func (h *MulticastSessions) Create(w http.ResponseWriter, r *http.Request) {
	var req createMulticastSessionRequest
	if !response.Decode(w, r, &req) {
		return
	}
	if req.ImageID == "" {
		response.BadRequest(w, "imageId is required")
		return
	}
	if len(req.HostIDs) == 0 {
		response.BadRequest(w, "at least one hostId is required")
		return
	}

	imageUUID, err := uuid.Parse(req.ImageID)
	if err != nil {
		response.BadRequest(w, "invalid imageId")
		return
	}

	portbase := req.Portbase
	if portbase <= 0 {
		portbase = 9000
	}

	// Create the multicast session.
	sess, err := h.db.MulticastSession.Create().
		SetName(req.Name).
		SetImageID(imageUUID).
		SetPortbase(portbase).
		SetClientCount(len(req.HostIDs)).
		Save(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}

	// Create a task for each host.
	for _, hostIDStr := range req.HostIDs {
		hostUUID, parseErr := uuid.Parse(hostIDStr)
		if parseErr != nil {
			continue
		}
		taskErr := h.db.Task.Create().
			SetType(enttask.TypeMulticast).
			SetState(enttask.StateQueued).
			SetHostID(hostUUID).
			SetImageID(imageUUID).
			SetMulticastSessionID(sess.ID).
			SetName(req.Name).
			Exec(r.Context())
		if taskErr != nil {
			// Log and continue; don't fail the whole request.
			_ = taskErr
		}
	}

	response.Created(w, sess)
}

func (h *MulticastSessions) List(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.db.MulticastSession.Query().All(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, response.ListOf(sessions))
}

func (h *MulticastSessions) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	sess, err := h.db.MulticastSession.Get(r.Context(), id)
	if err != nil {
		if ent.IsNotFound(err) {
			response.NotFound(w, "multicast session")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, sess)
}
