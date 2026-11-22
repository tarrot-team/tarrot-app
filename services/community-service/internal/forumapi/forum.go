package forumapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrPostNotFound = errors.New("post not found")
	ErrNoAuthor     = errors.New("no author")
)

type Role string

const sessionCookie = "session"

const (
	RoleUser      Role = "user"
	RoleAdmin     Role = "admin"
	RoleModerator Role = "moderator"
	RoleGuest     Role = "guest"
)

type User struct {
	ID   string
	Role Role
}

type Post struct {
	ID     string `json:"id"`
	Author string `json:"author_id"`
	Body   string `json:"body"`
}

type PostStore struct {
	mu    sync.RWMutex
	posts map[string]Post
}

func NewPostStore(posts ...Post) *PostStore {
	s := &PostStore{posts: make(map[string]Post, len(posts))}
	for _, post := range posts {
		s.posts[post.ID] = post
	}
	return s
}

func (s *PostStore) Get(id string) (Post, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	post, ok := s.posts[id]
	return post, ok
}

func (s *PostStore) UpdateBodyIfAuthor(id, userID, body string) (Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	post, ok := s.posts[id]
	if !ok {
		return Post{}, ErrPostNotFound
	}
	if post.Author != userID {
		return Post{}, ErrNoAuthor
	}
	post.Body = body
	s.posts[id] = post
	return post, nil
}

type session struct {
	user      User      //пользователь которому пренадлежит токен
	expiredAt time.Time //время окончания сессии
}

type Session struct {
	mu     sync.Mutex         //блокировка для доступа к словарю из нескольких запросов
	values map[string]session //ключ
	ttl    time.Duration      //сколько идет сессия
}

func NewSessions(ttl time.Duration) *Session {
	if ttl <= 0 {
		ttl = time.Hour * 24
	}
	return &Session{
		values: make(map[string]session),
		ttl:    ttl,
	}
}
func (s *Session) Create(user User) (string, error) {
	if user.ID == "" {
		return "", errors.New("user ID is not required")
	}
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[token] = session{
		user:      user,
		expiredAt: time.Now().Add(s.ttl),
	}
	return token, nil
}

func (s *Session) Revoke(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.values, token)
}

func (s *Session) Lookup(token string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[token]
	if !ok { //ищем сессию по токену. ok сообщает, есть ли такой ключ.
		return User{}, false
	}

	if !time.Now().Before(value.expiredAt) { // если срок уже истёк, удаляем сессию из словаря
		delete(s.values, token)
		return User{}, false
	}
	return value.user, true //токен найден и ещё действует
}

type Handler struct {
	posts    *PostStore
	sessions *Session
	content  *ContentStore
	mux      *http.ServeMux
}

func NewHandler(posts *PostStore, sessions *Session) *Handler {
	return NewHandlerWithContent(posts, sessions, NewContentStore(nil, nil, nil))
}

type userContextKey struct{}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if user, ok := h.sessions.Lookup(cookie.Value); ok {
			ctx := context.WithValue(r.Context(), userContextKey{}, user)
			r = r.WithContext(ctx)
		}
	}

	h.mux.ServeHTTP(w, r)
}

func currentUser(r *http.Request) (User, bool) {
	user, ok := r.Context().Value(userContextKey{}).(User)
	return user, ok
}

func readBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	var input struct {
		Body string `json:"body"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return "", false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		http.Error(w, "expected one JSON object", http.StatusBadRequest)
		return "", false
	}
	if strings.TrimSpace(input.Body) == "" {
		http.Error(w, "body is required", http.StatusBadRequest)
		return "", false
	}

	return input.Body, true
}

func (h *Handler) patchPost(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	body, ok := readBody(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	post, err := h.posts.UpdateBodyIfAuthor(id, user.ID, body)

	switch {
	case errors.Is(err, ErrPostNotFound):
		http.NotFound(w, r)
	case errors.Is(err, ErrNoAuthor):
		http.Error(w, "forbidden", http.StatusForbidden)
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(post)
	}
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if _, ok := currentUser(r); !ok { ///проверяем что ServeHTTP нашёл действующую сессию.
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	cookie, err := r.Cookie(sessionCookie) //получаем исходный токен из cookie, чтобы удалить именно эту сессию.
	if err != nil {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	h.sessions.Revoke(cookie.Value)
	http.SetCookie(w, &http.Cookie{ //отправляем браузеру указание удалить cookie.
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// 204 означает, что выход выполнен
	w.WriteHeader(http.StatusNoContent)
}
