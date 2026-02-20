package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ayushbhandari/event-api/internal/events"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const webDir = "web"

type Router struct {
	mux  *chi.Mux
	repo *events.Repository
}

func NewRouter(client *mongo.Client) http.Handler {
	db := client.Database("infosys")
	r := &Router{
		mux:  chi.NewRouter(),
		repo: events.NewRepository(db),
	}

	r.routes()
	return r.mux
}

func (r *Router) routes() {
	r.mux.Route("/api", func(api chi.Router) {
		api.Post("/users", r.handleCreateUser)
		api.Post("/login", r.handleLogin)

		api.Get("/events", r.handleListEvents)
		api.Post("/events", r.handleCreateEvent)
		api.Get("/events/{id}", r.handleGetEvent)
		api.Post("/events/{id}/registrations", r.handleRegisterForEvent)
		api.Get("/events/{id}/registrations", r.handleListRegistrations)
	})

	fs := http.FileServer(http.Dir(webDir))
	r.mux.Handle("/", fs)
	r.mux.Handle("/*", fs)
}

type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *Router) handleCreateUser(w http.ResponseWriter, req *http.Request) {
	var in createUserRequest
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if in.Name == "" || in.Email == "" || in.Password == "" {
		http.Error(w, "name, email and password are required", http.StatusBadRequest)
		return
	}
	u, err := r.repo.CreateUser(req.Context(), in.Name, in.Email, in.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	var in loginRequest
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	u, err := r.repo.GetUserByEmail(req.Context(), in.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if u == nil || u.Password != in.Password {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

type createEventRequest struct {
	OrganizerID string `json:"organizer_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Capacity    int    `json:"capacity"`
	StartsAt    string `json:"starts_at"`
	EndsAt      string `json:"ends_at"`
}

func (r *Router) handleCreateEvent(w http.ResponseWriter, req *http.Request) {
	var in createEventRequest
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	orgID, err := primitive.ObjectIDFromHex(in.OrganizerID)
	if err != nil {
		http.Error(w, "invalid organizer_id", http.StatusBadRequest)
		return
	}
	startsAt, err := time.Parse(time.RFC3339, in.StartsAt)
	if err != nil {
		http.Error(w, "invalid starts_at", http.StatusBadRequest)
		return
	}
	endsAt, err := time.Parse(time.RFC3339, in.EndsAt)
	if err != nil {
		http.Error(w, "invalid ends_at", http.StatusBadRequest)
		return
	}

	e, err := r.repo.CreateEvent(req.Context(), orgID, in.Title, in.Description, in.Capacity, startsAt, endsAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (r *Router) handleListEvents(w http.ResponseWriter, req *http.Request) {
	eventsList, err := r.repo.ListEvents(req.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, eventsList)
}

func (r *Router) handleGetEvent(w http.ResponseWriter, req *http.Request) {
	idStr := chi.URLParam(req, "id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	e, err := r.repo.GetEvent(req.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if e == nil {
		http.NotFound(w, req)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

type registerRequest struct {
	UserID string `json:"user_id"`
}

func (r *Router) handleRegisterForEvent(w http.ResponseWriter, req *http.Request) {
	idStr := chi.URLParam(req, "id")
	eventID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var in registerRequest
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	userID, err := primitive.ObjectIDFromHex(in.UserID)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	err = r.repo.RegisterForEvent(req.Context(), eventID, userID)
	if err != nil {
		switch err {
		case events.ErrCapacityFull:
			http.Error(w, err.Error(), http.StatusConflict)
			return
		case events.ErrAlreadyRegistered:
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
}

func (r *Router) handleListRegistrations(w http.ResponseWriter, req *http.Request) {
	idStr := chi.URLParam(req, "id")
	eventID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	regIDs, err := r.repo.ListRegistrations(req.Context(), eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, regIDs)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

