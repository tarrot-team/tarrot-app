package forumapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

var (
	ErrTopicNotFound   = errors.New("topic not found")
	ErrTopicClosed     = errors.New("topic is closed")
	ErrCommentNotFound = errors.New("comment not found")
	ErrArticleNotFound = errors.New("article not found")
	ErrForbidden       = errors.New("operation is forbidden")
)

type Topic struct {
	ID     string `json:"id"`
	Closed bool   `json:"closed"`
	Hidden bool   `json:"hidden"`
}

type Comment struct {
	ID       string `json:"id"`
	TopicID  string `json:"topic_id"`
	AuthorID string `json:"author_id"`
	Body     string `json:"body"`
}

type Article struct {
	ID        string `json:"id"`
	Published bool   `json:"published"`
}

type ModerationEvent struct {
	ActorID string    `json:"actor_id"`
	TopicID string    `json:"topic_id"`
	Action  string    `json:"action"`
	Time    time.Time `json:"time"`
}

type ContentStore struct {
	mu       sync.RWMutex
	topics   map[string]Topic
	comments map[string]Comment
	articles map[string]Article
	events   []ModerationEvent
}

func NewContentStore(topics []Topic, comments []Comment, articles []Article) *ContentStore {
	s := &ContentStore{
		topics:   make(map[string]Topic, len(topics)),
		comments: make(map[string]Comment, len(comments)),
		articles: make(map[string]Article, len(articles)),
	}
	for _, topic := range topics {
		s.topics[topic.ID] = topic
	}
	for _, comment := range comments {
		s.comments[comment.ID] = comment
	}
	for _, article := range articles {
		s.articles[article.ID] = article
	}
	return s
}

func (s *ContentStore) GetTopic(id string) (Topic, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	topic, ok := s.topics[id]
	return topic, ok
}

func (s *ContentStore) GetComment(id string) (Comment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	comment, ok := s.comments[id]
	return comment, ok
}

func (s *ContentStore) GetArticle(id string) (Article, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	article, ok := s.articles[id]
	return article, ok
}

func (s *ContentStore) ModerationLog() []ModerationEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]ModerationEvent(nil), s.events...)
}

func (s *ContentStore) CreateComment(topicID, authorID, body string) (Comment, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Comment{}, err
	}
	id := base64.RawURLEncoding.EncodeToString(raw[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	topic, ok := s.topics[topicID]
	if !ok {
		return Comment{}, ErrTopicNotFound
	}
	if topic.Closed {
		return Comment{}, ErrTopicClosed
	}
	comment := Comment{ID: id, TopicID: topicID, AuthorID: authorID, Body: body}
	s.comments[id] = comment
	return comment, nil
}

func (s *ContentStore) UpdateCommentIfAuthor(id, authorID, body string) (Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	comment, ok := s.comments[id]
	if !ok {
		return Comment{}, ErrCommentNotFound
	}
	if comment.AuthorID != authorID {
		return Comment{}, ErrNoAuthor
	}
	comment.Body = body
	s.comments[id] = comment
	return comment, nil
}

func (s *ContentStore) HideTopic(id string, actor User) (Topic, error) {
	if actor.Role != RoleModerator && actor.Role != RoleAdmin {
		return Topic{}, ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	topic, ok := s.topics[id]
	if !ok {
		return Topic{}, ErrTopicNotFound
	}
	if !topic.Hidden {
		topic.Hidden = true
		s.topics[id] = topic
		s.events = append(s.events, ModerationEvent{
			ActorID: actor.ID, TopicID: id, Action: "hide", Time: time.Now().UTC(),
		})
	}
	return topic, nil
}

func (s *ContentStore) PublishArticle(id string, actor User) (Article, error) {
	if actor.Role != RoleAdmin {
		return Article{}, ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	article, ok := s.articles[id]
	if !ok {
		return Article{}, ErrArticleNotFound
	}
	article.Published = true
	s.articles[id] = article
	return article, nil
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	body, ok := readBody(w, r)
	if !ok {
		return
	}
	comment, err := h.content.CreateComment(r.PathValue("id"), user.ID, body)
	switch {
	case errors.Is(err, ErrTopicNotFound):
		http.NotFound(w, r)
	case errors.Is(err, ErrTopicClosed):
		http.Error(w, "topic is closed", http.StatusConflict)
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Location", "/comments/"+comment.ID)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(comment)
	}
}

func (h *Handler) patchComment(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	body, ok := readBody(w, r)
	if !ok {
		return
	}
	comment, err := h.content.UpdateCommentIfAuthor(r.PathValue("id"), user.ID, body)
	switch {
	case errors.Is(err, ErrCommentNotFound):
		http.NotFound(w, r)
	case errors.Is(err, ErrNoAuthor):
		http.Error(w, "forbidden", http.StatusForbidden)
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(comment)
	}
}

func (h *Handler) hideTopic(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	topic, err := h.content.HideTopic(r.PathValue("id"), user)
	switch {
	case errors.Is(err, ErrForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, ErrTopicNotFound):
		http.NotFound(w, r)
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(topic)
	}
}

func (h *Handler) publishArticle(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	article, err := h.content.PublishArticle(r.PathValue("id"), user)
	switch {
	case errors.Is(err, ErrForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, ErrArticleNotFound):
		http.NotFound(w, r)
	case err != nil:
		http.Error(w, "internal error", http.StatusInternalServerError)
	default:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(article)
	}
}

func NewHandlerWithContent(posts *PostStore, sessions *Session, content *ContentStore) *Handler {
	h := &Handler{
		posts:    posts,
		sessions: sessions,
		content:  content,
		mux:      http.NewServeMux(),
	}
	h.mux.HandleFunc("PATCH /posts/{id}", h.patchPost)
	h.mux.HandleFunc("POST /logout", h.logout)
	h.mux.HandleFunc("POST /topics/{id}/comments", h.createComment)
	h.mux.HandleFunc("PATCH /comments/{id}", h.patchComment)
	h.mux.HandleFunc("POST /topics/{id}/hide", h.hideTopic)
	h.mux.HandleFunc("POST /articles/{id}/publish", h.publishArticle)
	return h
}
