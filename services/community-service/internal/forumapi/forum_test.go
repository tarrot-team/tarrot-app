package forumapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPatchPostPermissions(t *testing.T) {
	// Для каждого запроса задаём пользователя, ожидаемый статус и текст поста.
	tests := []struct {
		name       string
		user       *User
		wantStatus int
		wantBody   string
	}{
		{"гость", nil, http.StatusUnauthorized, "Старый текст"},
		{"автор", &User{ID: "alice", Role: RoleUser}, http.StatusOK, "Новый текст"},
		{"другой пользователь", &User{ID: "bob", Role: RoleUser}, http.StatusForbidden, "Старый текст"},
		{"модератор", &User{ID: "mod", Role: RoleModerator}, http.StatusForbidden, "Старый текст"},
		{"админ", &User{ID: "admin", Role: RoleAdmin}, http.StatusForbidden, "Старый текст"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Каждый сценарий начинает с одного и того же исходного поста.
			posts := NewPostStore(Post{
				ID:     "post-1",
				Author: "alice",
				Body:   "Старый текст",
			})
			sessions := NewSessions(time.Hour)
			handler := NewHandler(posts, sessions)

			// Формируем запрос на изменение текста поста.
			request := httptest.NewRequest(
				http.MethodPatch,
				"/posts/post-1",
				strings.NewReader(`{"body":"Новый текст"}`),
			)

			// Для гостя tt.user равен nil, поэтому cookie ему не добавляем.
			if tt.user != nil {
				token, err := sessions.Create(*tt.user)
				if err != nil {
					t.Fatalf("не удалось создать сессию: %v", err)
				}
				request.AddCookie(&http.Cookie{
					Name:  sessionCookie,
					Value: token,
				})
			}

			// Пропускаем запрос через тот же обработчик, который использует сервер.
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("статус %d, ожидали %d, ответ %q",
					response.Code, tt.wantStatus, response.Body.String())
			}

			// Читаем пост заново и проверяем, что попало в хранилище.
			post, ok := posts.Get("post-1")
			if !ok {
				t.Fatal("пост пропал из хранилища")
			}
			if post.Body != tt.wantBody {
				t.Fatalf("текст поста %q, ожидали %q", post.Body, tt.wantBody)
			}
		})
	}
}
func TestLogoutRevokesSession(t *testing.T) {
	posts := NewPostStore(Post{
		ID:     "post-1",
		Author: "alice",
		Body:   "Старый текст",
	})
	sessions := NewSessions(time.Hour)
	handler := NewHandler(posts, sessions)

	// Создаём действующую сессию для автора поста.
	token, err := sessions.Create(User{ID: "alice", Role: RoleUser})
	if err != nil {
		t.Fatalf("не удалось создать сессию: %v", err)
	}

	// Выходим с токеном Алисы.
	logoutRequest := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	logoutResponse := httptest.NewRecorder()
	handler.ServeHTTP(logoutResponse, logoutRequest)

	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("выход вернул %d, ожидали 204", logoutResponse.Code)
	}

	// Проверяем, что сервер попросил браузер удалить cookie.
	cookies := logoutResponse.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != sessionCookie || cookies[0].MaxAge != -1 {
		t.Fatalf("ожидали удаление cookie сессии, получили %v", cookies)
	}

	// Намеренно отправляем тот же токен в новом запросе.
	patchRequest := httptest.NewRequest(
		http.MethodPatch,
		"/posts/post-1",
		strings.NewReader(`{"body":"Новый текст"}`),
	)
	patchRequest.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	patchResponse := httptest.NewRecorder()
	handler.ServeHTTP(patchResponse, patchRequest)

	if patchResponse.Code != http.StatusUnauthorized {
		t.Fatalf("запрос после выхода вернул %d, ожидали 401", patchResponse.Code)
	}

	post, ok := posts.Get("post-1")
	if !ok || post.Body != "Старый текст" {
		t.Fatalf("после отклонённого запроса пост = %+v, найден = %t", post, ok)
	}
}

func TestCreateComment(t *testing.T) {
	content := NewContentStore([]Topic{{ID: "topic-1"}}, nil, nil)
	sessions := NewSessions(time.Hour)
	handler := NewHandlerWithContent(NewPostStore(), sessions, content)

	// Создаём сессию Алисы, чтобы отправить запрос от её имени.
	token, err := sessions.Create(User{ID: "alice", Role: RoleUser})
	if err != nil {
		t.Fatalf("не удалось создать сессию: %v", err)
	}

	// Собираем POST-запрос с текстом комментария и добавляем cookie.
	request := httptest.NewRequest(
		http.MethodPost,
		"/topics/topic-1/comments",
		strings.NewReader(`{"body":"Мой комментарий"}`),
	)
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})

	// Передаём запрос обработчику и проверяем статус 201.
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("статус = %d, ожидали 201, ответ = %q",
			response.Code, response.Body.String())
	}

	// Читаем JSON ответа, чтобы узнать ID созданного комментария.
	var comment Comment
	if err := json.Unmarshal(response.Body.Bytes(), &comment); err != nil {
		t.Fatalf("не удалось прочитать JSON ответа: %v", err)
	}

	// По ID из ответа ищем комментарий в хранилище и сверяем его поля.
	stored, ok := content.GetComment(comment.ID)
	if !ok {
		t.Fatalf("комментарий %q не найден в хранилище", comment.ID)
	}
	if stored.TopicID != "topic-1" || stored.AuthorID != "alice" || stored.Body != "Мой комментарий" {
		t.Fatalf("сохранён неожиданный комментарий: %+v", stored)
	}
}

func TestCreateCommentRejected(t *testing.T) {
	tests := []struct {
		name       string
		closed     bool
		loggedIn   bool
		wantStatus int
	}{
		{name: "гость", wantStatus: http.StatusUnauthorized},
		{name: "закрытая тема", closed: true, loggedIn: true, wantStatus: http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Каждый сценарий получает свою тему и пустое хранилище комментариев.
			content := NewContentStore(
				[]Topic{{ID: "topic-1", Closed: tt.closed}},
				nil,
				nil,
			)
			sessions := NewSessions(time.Hour)
			handler := NewHandlerWithContent(NewPostStore(), sessions, content)

			request := httptest.NewRequest(
				http.MethodPost,
				"/topics/topic-1/comments",
				strings.NewReader(`{"body":"Мой комментарий"}`),
			)

			// Прикладываем cookie только для вошедшего пользователя.
			if tt.loggedIn {
				token, err := sessions.Create(User{ID: "alice", Role: RoleUser})
				if err != nil {
					t.Fatalf("не удалось создать сессию: %v", err)
				}
				request.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидали %d, ответ = %q",
					response.Code, tt.wantStatus, response.Body.String())
			}

			// Одного статуса мало. Проверяем, что сервер ничего не сохранил.
			if count := len(content.comments); count != 0 {
				t.Fatalf("после отказа в хранилище %d комментариев, ожидали 0", count)
			}
		})
	}
}

func TestPatchCommentPermissions(t *testing.T) {
	tests := []struct {
		name       string
		user       User
		wantStatus int
		wantBody   string
	}{
		{
			name:       "автор",
			user:       User{ID: "alice", Role: RoleUser},
			wantStatus: http.StatusOK,
			wantBody:   "Новый текст",
		},
		{
			name:       "другой пользователь",
			user:       User{ID: "bob", Role: RoleUser},
			wantStatus: http.StatusForbidden,
			wantBody:   "Старый текст",
		},
		{
			name:       "модератор",
			user:       User{ID: "mod", Role: RoleModerator},
			wantStatus: http.StatusForbidden,
			wantBody:   "Старый текст",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Каждый сценарий начинает с одного и того же комментария Алисы.
			content := NewContentStore(
				[]Topic{{ID: "topic-1"}},
				[]Comment{{
					ID:       "comment-1",
					TopicID:  "topic-1",
					AuthorID: "alice",
					Body:     "Старый текст",
				}},
				nil,
			)
			sessions := NewSessions(time.Hour)
			handler := NewHandlerWithContent(NewPostStore(), sessions, content)

			// Создаём сессию для пользователя из текущего сценария.
			token, err := sessions.Create(tt.user)
			if err != nil {
				t.Fatalf("не удалось создать сессию: %v", err)
			}

			request := httptest.NewRequest(
				http.MethodPatch,
				"/comments/comment-1",
				strings.NewReader(`{"body":"Новый текст"}`),
			)
			request.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидали %d, ответ = %q",
					response.Code, tt.wantStatus, response.Body.String())
			}

			// Читаем комментарий заново, чтобы проверить сохранённый текст.
			stored, ok := content.GetComment("comment-1")
			if !ok {
				t.Fatal("комментарий не найден в хранилище")
			}
			if stored.Body != tt.wantBody {
				t.Fatalf("текст = %q, ожидали %q", stored.Body, tt.wantBody)
			}
		})
	}
}

func TestHideTopicPermissionsAndAudit(t *testing.T) {
	tests := []struct {
		name       string
		user       *User
		wantStatus int
		wantHidden bool
	}{
		{"гость", nil, http.StatusUnauthorized, false},
		{"обычный пользователь", &User{ID: "alice", Role: RoleUser}, http.StatusForbidden, false},
		{"модератор", &User{ID: "mod", Role: RoleModerator}, http.StatusOK, true},
		{"админ", &User{ID: "admin", Role: RoleAdmin}, http.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Каждый сценарий начинает с видимой темы и пустого журнала.
			const topicID = "topic-1"
			content := NewContentStore([]Topic{{ID: topicID}}, nil, nil)
			sessions := NewSessions(time.Hour)
			handler := NewHandlerWithContent(NewPostStore(), sessions, content)

			request := httptest.NewRequest(
				http.MethodPost,
				"/topics/"+topicID+"/hide",
				nil,
			)

			// Гость отправляет запрос без cookie.
			if tt.user != nil {
				token, err := sessions.Create(*tt.user)
				if err != nil {
					t.Fatalf("не удалось создать сессию: %v", err)
				}
				request.AddCookie(&http.Cookie{
					Name:  sessionCookie,
					Value: token,
				})
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидали %d, ответ = %q",
					response.Code, tt.wantStatus, response.Body.String())
			}

			// Проверяем состояние темы после запроса.
			topic, ok := content.GetTopic(topicID)
			if !ok {
				t.Fatal("тема не найдена в хранилище")
			}
			if topic.Hidden != tt.wantHidden {
				t.Fatalf("Hidden = %t, ожидали %t", topic.Hidden, tt.wantHidden)
			}

			events := content.ModerationLog()
			if !tt.wantHidden {
				if len(events) != 0 {
					t.Fatalf("после отказа появились записи в журнале: %+v", events)
				}
				return
			}

			// После успешного скрытия проверяем все поля записи.
			if len(events) != 1 {
				t.Fatalf("записей в журнале = %d, ожидали 1", len(events))
			}

			event := events[0]
			if event.ActorID != tt.user.ID ||
				event.TopicID != topicID ||
				event.Action != "hide" ||
				event.Time.IsZero() {
				t.Fatalf("неверная запись в журнале: %+v", event)
			}
		})
	}
}
func TestPublishArticlePermissions(t *testing.T) {
	tests := []struct {
		name          string
		user          *User
		wantStatus    int
		wantPublished bool
	}{
		{"гость", nil, http.StatusUnauthorized, false},
		{"пользователь", &User{ID: "alice", Role: RoleUser}, http.StatusForbidden, false},
		{"модератор", &User{ID: "mod", Role: RoleModerator}, http.StatusForbidden, false},
		{"админ", &User{ID: "admin", Role: RoleAdmin}, http.StatusOK, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Каждый сценарий начинает с одной неопубликованной статьи.
			const articleID = "article-1"
			content := NewContentStore(nil, nil, []Article{{ID: articleID}})
			sessions := NewSessions(time.Hour)
			handler := NewHandlerWithContent(NewPostStore(), sessions, content)

			// Для публикации достаточно ID статьи в URL.
			request := httptest.NewRequest(
				http.MethodPost,
				"/articles/"+articleID+"/publish",
				nil,
			)

			// У гостя tt.user равен nil, поэтому его запрос идёт без cookie.
			if tt.user != nil {
				token, err := sessions.Create(*tt.user)
				if err != nil {
					t.Fatalf("не удалось создать сессию: %v", err)
				}
				request.AddCookie(&http.Cookie{
					Name:  sessionCookie,
					Value: token,
				})
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидали %d, ответ = %q",
					response.Code, tt.wantStatus, response.Body.String())
			}

			// Читаем статью из хранилища, чтобы проверить результат запроса.
			article, ok := content.GetArticle(articleID)
			if !ok {
				t.Fatal("статья не найдена в хранилище")
			}
			if article.Published != tt.wantPublished {
				t.Fatalf("Published = %t, ожидали %t",
					article.Published, tt.wantPublished)
			}
		})
	}
}
