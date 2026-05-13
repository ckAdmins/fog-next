package handlers

import (
"net/http"

"github.com/go-chi/chi/v5"
"github.com/nemvince/fog-next/ent"
entuser "github.com/nemvince/fog-next/ent/user"
"github.com/nemvince/fog-next/internal/api/response"
fogauth "github.com/nemvince/fog-next/internal/auth"
"github.com/nemvince/fog-next/internal/config"
)

type Users struct {
cfg *config.Config
db  *ent.Client
}

func NewUsers(cfg *config.Config, db *ent.Client) *Users { return &Users{cfg, db} }

func (h *Users) List(w http.ResponseWriter, r *http.Request) {
users, err := h.db.User.Query().Limit(200).All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(users))
}

func (h *Users) Get(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
u, err := h.db.User.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "user")
return
}
response.InternalError(w)
return
}
response.OK(w, u)
}

func (h *Users) Create(w http.ResponseWriter, r *http.Request) {
var body struct {
Username string         `json:"username"`
Password string         `json:"password"`
Role     entuser.Role   `json:"role"`
Email    string         `json:"email"`
}
if !response.Decode(w, r, &body) {
return
}
if body.Username == "" || body.Password == "" {
response.BadRequest(w, "username and password are required")
return
}
hash, err := fogauth.HashPassword(body.Password)
if err != nil {
response.InternalError(w)
return
}
if body.Role == "" {
body.Role = entuser.RoleReadonly
}
u, err := h.db.User.Create().
SetUsername(body.Username).
SetPasswordHash(hash).
SetRole(body.Role).
SetEmail(body.Email).
SetIsActive(true).
Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
writeAudit(r.Context(), h.db, r, "create", "user", u.ID.String(), u.Username)
response.Created(w, u)
}

func (h *Users) Update(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
existing, err := h.db.User.Get(r.Context(), id)
if err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "user")
return
}
response.InternalError(w)
return
}
var body struct {
Email    string       `json:"email"`
Role     entuser.Role `json:"role"`
IsActive bool         `json:"isActive"`
Password string       `json:"password"`
}
if !response.Decode(w, r, &body) {
return
}
up := h.db.User.UpdateOneID(id).
SetEmail(body.Email).
SetIsActive(body.IsActive)
if body.Role != "" {
up = up.SetRole(body.Role)
}
if body.Password != "" {
hash, err := fogauth.HashPassword(body.Password)
if err != nil {
response.InternalError(w)
return
}
up = up.SetPasswordHash(hash)
}
_ = existing // used for 404 check above
updated, err := up.Save(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, updated)
}

func (h *Users) Delete(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if err := h.db.User.DeleteOneID(id).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
writeAudit(r.Context(), h.db, r, "delete", "user", id.String(), "")
response.NoContent(w)
}

func (h *Users) RegenerateToken(w http.ResponseWriter, r *http.Request) {
id, ok := parseUUID(w, chi.URLParam(r, "id"))
if !ok {
return
}
if _, err := h.db.User.Get(r.Context(), id); err != nil {
if ent.IsNotFound(err) {
response.NotFound(w, "user")
return
}
response.InternalError(w)
return
}
token, err := fogauth.GenerateAPIToken()
if err != nil {
response.InternalError(w)
return
}
if err := h.db.User.UpdateOneID(id).SetAPIToken(token).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.OK(w, map[string]string{"apiToken": token})
}
