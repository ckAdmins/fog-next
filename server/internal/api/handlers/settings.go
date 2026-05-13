package handlers

import (
"net/http"

"github.com/go-chi/chi/v5"
"github.com/nemvince/fog-next/ent"
"github.com/nemvince/fog-next/ent/globalsetting"
"github.com/nemvince/fog-next/internal/api/response"
)

type Settings struct{ db *ent.Client }

func NewSettings(db *ent.Client) *Settings { return &Settings{db} }

func (h *Settings) List(w http.ResponseWriter, r *http.Request) {
cat := r.URL.Query().Get("category")
query := h.db.GlobalSetting.Query()
if cat != "" {
query = query.Where(globalsetting.CategoryEQ(cat))
}
settings, err := query.All(r.Context())
if err != nil {
response.InternalError(w)
return
}
response.OK(w, response.ListOf(settings))
}

func (h *Settings) Set(w http.ResponseWriter, r *http.Request) {
key := chi.URLParam(r, "key")
var body struct {
Value    string `json:"value"`
Category string `json:"category"`
}
if !response.Decode(w, r, &body) {
return
}
if err := h.db.GlobalSetting.Create().
SetKey(key).
SetValue(body.Value).
SetNillableCategory(&body.Category).
OnConflictColumns(globalsetting.FieldKey).
UpdateNewValues().
Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}

func (h *Settings) Delete(w http.ResponseWriter, r *http.Request) {
key := chi.URLParam(r, "key")
if _, err := h.db.GlobalSetting.Delete().Where(globalsetting.KeyEQ(key)).Exec(r.Context()); err != nil {
response.InternalError(w)
return
}
response.NoContent(w)
}
