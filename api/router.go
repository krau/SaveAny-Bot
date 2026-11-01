package api

import (
	"net/http"
	"time"

	"github.com/charmbracelet/log"

	"github.com/krau/SaveAny-Bot/config"
	"github.com/krau/SaveAny-Bot/core"
)

type server struct {
	deps                   Deps
	factory                *TaskFactory
	mediaMetadataExtractor mediaMetadataExtractor
	logins                 *loginLimiter
	startedAt              time.Time
}

// NewHandler builds the redesigned JSON API mounted under /api/v1.
func NewHandler(deps Deps) http.Handler {
	s := &server{
		deps:                   deps,
		factory:                NewTaskFactory(deps.Ctx),
		mediaMetadataExtractor: extractMediaMetadata,
		logins:                 newLoginLimiter(),
		startedAt:              time.Now(),
	}
	go s.sessionCleanup(deps.Ctx)
	StartCleanupLoop(deps.Ctx)
	core.SetTaskSinkFactory(TaskProgressSink)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /api/v1/auth/login", s.login)
	mux.HandleFunc("GET /api/v1/auth/token", s.tokenLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)
	mux.HandleFunc("GET /api/v1/auth/me", s.me)
	mux.HandleFunc("POST /api/v1/auth/password", s.changePassword)
	mux.HandleFunc("GET /api/v1/onboarding", s.onboarding)

	mux.HandleFunc("GET /api/v1/settings", s.getSettings)
	mux.HandleFunc("PUT /api/v1/settings/{group}", s.updateSettings)

	mux.HandleFunc("GET /api/v1/storages", s.listStorages)
	mux.HandleFunc("POST /api/v1/storages", s.createStorage)
	mux.HandleFunc("PUT /api/v1/storages/{id}", s.updateStorage)
	mux.HandleFunc("DELETE /api/v1/storages/{id}", s.deleteStorage)
	mux.HandleFunc("POST /api/v1/storages/{id}/test", s.testStorage)

	mux.HandleFunc("GET /api/v1/users", s.listUsers)
	mux.HandleFunc("POST /api/v1/users", s.createUser)
	mux.HandleFunc("PUT /api/v1/users/{id}", s.updateUser)
	mux.HandleFunc("DELETE /api/v1/users/{id}", s.deleteUser)

	mux.HandleFunc("GET /api/v1/telegram/status", s.telegramStatus)
	mux.HandleFunc("GET /api/v1/telegram/workers", s.listBotWorkers)
	mux.HandleFunc("POST /api/v1/telegram/workers", s.createBotWorker)
	mux.HandleFunc("PUT /api/v1/telegram/workers/{id}", s.updateBotWorker)
	mux.HandleFunc("DELETE /api/v1/telegram/workers/{id}", s.deleteBotWorker)
	mux.HandleFunc("POST /api/v1/telegram/workers/{id}/start", s.startBotWorker)
	mux.HandleFunc("POST /api/v1/telegram/workers/{id}/stop", s.stopBotWorker)
	mux.HandleFunc("POST /api/v1/telegram/workers/{id}/test", s.testBotWorker)
	mux.HandleFunc("POST /api/v1/telegram/bot/start", s.startBot)
	mux.HandleFunc("POST /api/v1/telegram/bot/stop", s.stopBot)
	mux.HandleFunc("POST /api/v1/telegram/bot/restart", s.restartBot)
	mux.HandleFunc("POST /api/v1/telegram/userbot/login", s.startUserbotLogin)
	mux.HandleFunc("POST /api/v1/telegram/userbot/login/{flow}/code", s.submitUserbotCode)
	mux.HandleFunc("POST /api/v1/telegram/userbot/login/{flow}/password", s.submitUserbotPassword)
	mux.HandleFunc("DELETE /api/v1/telegram/userbot/login/{flow}", s.cancelUserbotLogin)
	mux.HandleFunc("POST /api/v1/telegram/userbot/logout", s.logoutUserbot)

	mux.HandleFunc("GET /api/v1/tasks", s.listTasks)
	mux.HandleFunc("POST /api/v1/tasks", s.createTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}", s.getTask)
	mux.HandleFunc("POST /api/v1/tasks/{id}/cancel", s.cancelTask)
	mux.HandleFunc("POST /api/v1/tasks/{id}/retry", s.retryTask)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", s.deleteTask)
	mux.HandleFunc("GET /api/v1/task-types", s.taskTypes)
	mux.HandleFunc("GET /api/v1/media-metadata", s.mediaMetadata)

	mux.HandleFunc("GET /api/v1/system/info", s.systemInfo)
	mux.HandleFunc("POST /api/v1/system/restart", s.restartSystem)
	mux.HandleFunc("GET /api/v1/filesystem", s.listFilesystem)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, http.StatusNotFound, "not_found", "endpoint not found: "+r.URL.Path)
	})

	var handler http.Handler = mux
	handler = s.authMiddleware(handler)
	handler = loggingMiddleware(handler)
	handler = recoveryMiddleware(handler)
	return handler
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": config.Version})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		log.Infof("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("Panic recovered: %v", err)
				WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
