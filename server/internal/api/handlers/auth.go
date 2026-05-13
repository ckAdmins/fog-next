// Package handlers provides HTTP handler types, one per domain resource.
package handlers

import (
"crypto/sha256"
"fmt"
"net/http"
"time"

"github.com/google/uuid"
"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/ent/refreshtoken"
entuser "github.com/nemvince/fog-next/ent/user"
"github.com/nemvince/fog-next/internal/api/response"
fogauth "github.com/nemvince/fog-next/internal/auth"
"github.com/nemvince/fog-next/internal/config"
)

// Auth handles login, token refresh, and logout.
type Auth struct {
cfg *config.Config
db  *ent.Client
}

func NewAuth(cfg *config.Config, db *ent.Client) *Auth {
return &Auth{cfg: cfg, db: db}
}

type loginRequest struct {
Username string `json:"username"`
Password string `json:"password"`
}

type tokenResponse struct {
AccessToken  string    `json:"accessToken"`
RefreshToken string    `json:"refreshToken"`
ExpiresAt    time.Time `json:"expiresAt"`
}

func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
var req loginRequest
if !response.Decode(w, r, &req) {
return
}
if req.Username == "" || req.Password == "" {
response.BadRequest(w, "username and password are required")
return
}

u, err := h.db.User.Query().Where(entuser.UsernameEQ(req.Username)).Only(r.Context())
if err != nil {
if ent.IsNotFound(err) {
response.Error(w, http.StatusUnauthorized, "Unauthorized", "invalid credentials")
return
}
response.InternalError(w)
return
}

if !u.IsActive {
response.Error(w, http.StatusUnauthorized, "Unauthorized", "account disabled")
return
}

if err := fogauth.CheckPassword(u.PasswordHash, req.Password); err != nil {
response.Error(w, http.StatusUnauthorized, "Unauthorized", "invalid credentials")
return
}

pair, err := fogauth.IssueTokenPair(h.cfg.Auth, u.ID, u.Username, string(u.Role))
if err != nil {
response.InternalError(w)
return
}

hashBytes := sha256.Sum256([]byte(pair.RefreshToken))
tokenHash := fmt.Sprintf("%x", hashBytes)
if err := h.db.RefreshToken.Create().
SetUserID(u.ID).
SetTokenHash(tokenHash).
SetExpiresAt(time.Now().Add(h.cfg.Auth.RefreshTokenExpiry)).
Exec(r.Context()); err != nil {
response.InternalError(w)
return
}

_ = h.db.User.UpdateOneID(u.ID).SetLastLoginAt(time.Now()).Exec(r.Context())

response.OK(w, tokenResponse{
AccessToken:  pair.AccessToken,
RefreshToken: pair.RefreshToken,
ExpiresAt:    pair.ExpiresAt,
})
}

type refreshRequest struct {
RefreshToken string `json:"refreshToken"`
}

func (h *Auth) Refresh(w http.ResponseWriter, r *http.Request) {
var req refreshRequest
if !response.Decode(w, r, &req) {
return
}

hashBytes := sha256.Sum256([]byte(req.RefreshToken))
tokenHash := fmt.Sprintf("%x", hashBytes)

rt, err := h.db.RefreshToken.Query().Where(
refreshtoken.TokenHashEQ(tokenHash),
refreshtoken.RevokedAtIsNil(),
refreshtoken.ExpiresAtGT(time.Now()),
).Only(r.Context())
if err != nil {
response.Error(w, http.StatusUnauthorized, "Unauthorized", "invalid or expired refresh token")
return
}

u, err := h.db.User.Get(r.Context(), rt.UserID)
if err != nil || !u.IsActive {
response.Error(w, http.StatusUnauthorized, "Unauthorized", "user not found or inactive")
return
}

_ = h.db.RefreshToken.UpdateOneID(rt.ID).SetRevokedAt(time.Now()).Exec(r.Context())

pair, err := fogauth.IssueTokenPair(h.cfg.Auth, u.ID, u.Username, string(u.Role))
if err != nil {
response.InternalError(w)
return
}

newHash := sha256.Sum256([]byte(pair.RefreshToken))
if err := h.db.RefreshToken.Create().
SetUserID(u.ID).
SetTokenHash(fmt.Sprintf("%x", newHash)).
SetExpiresAt(time.Now().Add(h.cfg.Auth.RefreshTokenExpiry)).
Exec(r.Context()); err != nil {
response.InternalError(w)
return
}

response.OK(w, tokenResponse{
AccessToken:  pair.AccessToken,
RefreshToken: pair.RefreshToken,
ExpiresAt:    pair.ExpiresAt,
})
}

func (h *Auth) Logout(w http.ResponseWriter, r *http.Request) {
var req refreshRequest
_ = response.Decode(w, r, &req)

if req.RefreshToken != "" {
hashBytes := sha256.Sum256([]byte(req.RefreshToken))
tokenHash := fmt.Sprintf("%x", hashBytes)
rt, err := h.db.RefreshToken.Query().Where(
refreshtoken.TokenHashEQ(tokenHash),
refreshtoken.RevokedAtIsNil(),
).Only(r.Context())
if err == nil {
_ = h.db.RefreshToken.UpdateOneID(rt.ID).SetRevokedAt(time.Now()).Exec(r.Context())
}
}
response.NoContent(w)
}

// ── Shared helpers ──────────────────────────────────────────────────────────

// parseUUID parses a UUID path parameter, writing a 400 on failure.
func parseUUID(w http.ResponseWriter, s string) (uuid.UUID, bool) {
id, err := uuid.Parse(s)
if err != nil {
response.BadRequest(w, "invalid id: "+s)
return uuid.Nil, false
}
return id, true
}
