package api

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"algokanbana/internal/auth"
	"algokanbana/internal/config"
	"algokanbana/internal/lms"
	"algokanbana/internal/store"
	"algokanbana/internal/sync"

	"github.com/go-chi/chi/v5"
)

type ctxKey string

const sessionCtxKey ctxKey = "session_claims"

type Server struct {
	cfg     config.Config
	store   *store.Store
	auth    *auth.Manager
	lms     *lms.Client
	syncEng *sync.Engine
	sse     *SSEHub
	webFS   fs.FS
}

func NewServer(cfg config.Config, st *store.Store, webFS fs.FS) *Server {
	_ = os.MkdirAll(cfg.UploadDir, 0o755)
	sse := NewSSEHub()
	timeout := time.Duration(cfg.UpstreamTimeoutMs) * time.Millisecond
	s := &Server{
		cfg:   cfg,
		store: st,
		auth:  auth.NewManager(cfg.SessionSecret, cfg.SessionCookieName, cfg.CookieSameSite),
		lms:   lms.NewClient(cfg.LMSBaseURL, cfg.SafeReadonlyUpstream, timeout),
		sse:   sse,
		webFS: webFS,
	}
	s.syncEng = sync.NewEngine(cfg, sse.Broadcast)
	return s
}

func (s *Server) StartBackground(ctx context.Context) {
	s.syncEng.Start(ctx)
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(timingAndSecurityMiddleware)
	r.Use(gzipMiddleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "mode": s.cfg.AppMode})
	})

	// Uploaded proof images / PDFs
	fileServer := http.StripPrefix("/uploads/", http.FileServer(http.Dir(s.cfg.UploadDir)))
	r.Get("/uploads/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		fileServer.ServeHTTP(w, r)
	})

	r.Route("/api/v1", func(api chi.Router) {
		// Public auth endpoints
		api.Post("/auth/login", s.handleLogin)
		api.Post("/auth/logout", s.handleLogout)
		api.Get("/auth/me", s.handleMe)

		// Protected Tutor & Team Lead endpoints
		api.Group(func(priv chi.Router) {
			priv.Use(s.requireAuth)

			priv.Post("/auth/role", s.handleSwitchRole)
			priv.Get("/bootstrap", s.handleBootstrap)
			priv.Get("/events/stream", s.sse.ServeHTTP)
			priv.Post("/sync/trigger", s.handleTriggerSync)
			priv.Get("/search", s.handleSearch)
			priv.Post("/saved-views", s.handleCreateSavedView)
			priv.Delete("/saved-views/{id}", s.handleDeleteSavedView)
			priv.Post("/uploads", s.handleUploadFiles)

			// Dashboard & Students
			priv.Get("/dashboard/groups", s.handleDashboardGroups)
			priv.Get("/dashboard/groups/{id}", s.handleGroupDetail)
			priv.Get("/dashboard/kpi", s.handleDashboardKPI)
			priv.Get("/reports/tl", s.handleListTLReports)
			priv.Post("/reports/tl", s.handleCreateTLReport)
			priv.Get("/students", s.handleListStudents)
			priv.Get("/students/{id}", s.handleStudentDetail)

			// Taskboard & Urgent Substitute
			priv.Get("/tasks", s.handleListTasks)
			priv.Post("/tasks", s.handleCreateTask)
			priv.Patch("/tasks/bulk", s.handleBulkUpdateTasks)
			priv.Patch("/tasks/{id}/move", s.handleMoveTask)
			priv.Get("/urgent-subs", s.handleGetUrgentSubs)
			priv.Post("/urgent-subs", s.handleCreateUrgentSub)
			priv.Delete("/urgent-subs/{id}", s.handleWithdrawUrgentSub)

			// Weekly Routines & Student Surveys
			priv.Get("/routines/summary", s.handleGetRoutinesSummary)
			priv.Get("/surveys", s.handleListSurveys)
			priv.Get("/surveys/{id}", s.handleGetSurvey)
			priv.Post("/surveys/{id}/submit", s.handleSubmitSurvey)

			// Regular Feedback & Feedback MT
			priv.Get("/feedback/regular", s.handleGetRegularFeedback)
			priv.Post("/feedback/regular/preview", s.handlePreviewRegularFeedback)
			priv.Put("/feedback/regular", s.handleSaveRegularFeedback)
			priv.Get("/feedback/regular/{id}/print", s.handlePrintRegularFeedback)
			priv.Get("/feedback/mt", s.handleListMTFeedback)
			priv.Post("/feedback/mt", s.handleCreateMTFeedback)

			// Schedule, 217-Slot Availability, Reschedule & Workload
			priv.Get("/schedule", s.handleGetSchedule)
			priv.Put("/schedule/availability", s.handleUpdateAvailability)
			priv.Post("/reschedules", s.handleCreateReschedule)
			priv.Delete("/reschedules/{id}", s.handleWithdrawReschedule)
			priv.Post("/workload-requests", s.handleCreateWorkloadRequest)

			// Assessments
			priv.Get("/assessments", s.handleGetAssessments)
			priv.Post("/assessments/{cycle}/submit", s.handleSubmitAssessment)

			// Knowledge Base & Communication Threads
			priv.Get("/kb/articles", s.handleListKBArticles)
			priv.Get("/threads", s.handleListThreads)
			priv.Get("/threads/{id}", s.handleGetThread)
			priv.Post("/requests", s.handleCreateSupportRequest)
			priv.Post("/threads/{id}/messages", s.handleReplyThread)
			priv.Post("/threads/{id}/read", s.handleMarkThreadRead)
			priv.Post("/threads/{id}/escalate", s.handleEscalateThread)

			// Profile & Telegram
			priv.Get("/profile", s.handleGetProfile)
			priv.Patch("/profile", s.handleUpdateProfile)
			priv.Post("/profile/telegram", s.handleTelegramAction)

			// Team Lead Demo Queue & Approvals + Live Pitching Benchmark
			priv.Get("/tl/queue", s.handleTLDemoQueue)
			priv.Post("/tl/reschedules/{id}/review", s.handleTLReviewReschedule)
			priv.Post("/tl/reports/{id}/review", s.handleTLReviewReport)
			priv.Post("/tl/urgent-subs/{id}/review", s.handleTLReviewUrgentSub)
			priv.Get("/benchmark", s.handleBenchmark)
		})
	})

	// Embedded SPA Static + History API Fallback
	r.NotFound(s.serveSPA)
	return r
}

func (s *Server) serveSPA(w http.ResponseWriter, r *http.Request) {
	if s.webFS == nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("AlgoKanbana API Server is running. Build web/dist to serve UI."))
		return
	}

	reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if reqPath == "" || reqPath == "." {
		reqPath = "index.html"
	}

	if f, err := s.webFS.Open(reqPath); err == nil {
		stat, errStat := f.Stat()
		_ = f.Close()
		if errStat == nil && !stat.IsDir() {
			if strings.HasPrefix(reqPath, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else if reqPath == "index.html" {
				w.Header().Set("Cache-Control", "no-cache")
			}
			http.FileServer(http.FS(s.webFS)).ServeHTTP(w, r)
			return
		}
	}

	// SPA client-side route fallback -> serve index.html
	indexBytes, err := fs.ReadFile(s.webFS, "index.html")
	if err != nil {
		http.Error(w, "UI assets not built yet", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(indexBytes)
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := s.auth.FromRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "sesi berakhir atau belum login")
			return
		}
		ctx := context.WithValue(r.Context(), sessionCtxKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getClaims(r *http.Request) auth.SessionClaims {
	v, _ := r.Context().Value(sessionCtxKey).(auth.SessionClaims)
	return v
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
	start       time.Time
}

func (sr *statusRecorder) WriteHeader(code int) {
	if !sr.wroteHeader {
		sr.statusCode = code
		ms := float64(time.Since(sr.start).Microseconds()) / 1000.0
		sr.Header().Set("X-Response-Time-Ms", fmt.Sprintf("%.2f", ms))
		sr.wroteHeader = true
	}
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	if !sr.wroteHeader {
		sr.WriteHeader(http.StatusOK)
	}
	return sr.ResponseWriter.Write(b)
}

func (sr *statusRecorder) Flush() {
	if fl, ok := sr.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func timingAndSecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Access-Control-Expose-Headers", "X-Response-Time-Ms")
		rec := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK, start: time.Now()}
		next.ServeHTTP(rec, r)

		duration := time.Since(rec.start)
		ms := float64(duration.Microseconds()) / 1000.0
		if rec.statusCode >= 400 {
			log.Printf("[HTTP ERR] %s %s -> %d (%.2f ms)", r.Method, r.URL.Path, rec.statusCode, ms)
		} else if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/v1/events/stream" {
			log.Printf("[HTTP] %s %s -> %d (%.2f ms)", r.Method, r.URL.Path, rec.statusCode, ms)
		}
	})
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}

func (g *gzipResponseWriter) Flush() {
	if gw, ok := g.Writer.(*gzip.Writer); ok {
		_ = gw.Flush()
	}
	if fl, ok := g.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/events/stream" || !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		gw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
		defer func() { _ = gw.Close() }()
		next.ServeHTTP(&gzipResponseWriter{Writer: gw, ResponseWriter: w}, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	log.Printf("[API ERR] %d: %s", code, msg)
	writeJSON(w, code, map[string]any{"ok": false, "error": msg})
}

func decodeJSON(r *http.Request, dst any) error {
	defer func() { _ = r.Body.Close() }()
	dec := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	return dec.Decode(dst)
}
