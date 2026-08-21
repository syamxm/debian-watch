package httpx

import (
	"log/slog"
	"net/http"

	"github.com/syamxm/debian-watch/internal/auth"
	"github.com/syamxm/debian-watch/internal/config"
	"github.com/syamxm/debian-watch/internal/monitor"
	"github.com/syamxm/debian-watch/web"
)

type Server struct {
	cfg      config.Config
	assetVer string
	log      *slog.Logger
	renderer *Renderer
	monitor  *monitor.Monitor
	sessions *auth.SessionStore
	limiter  *auth.LoginLimiter
	creds    auth.Credentials
}

func NewServer(
	cfg config.Config,
	log *slog.Logger,
	metrics *monitor.Monitor,
	sessions *auth.SessionStore,
	limiter *auth.LoginLimiter,
	creds auth.Credentials,
) (*Server, error) {
	renderer, err := NewRenderer(web.Files)
	if err != nil {
		return nil, err
	}
	version, err := assetVersion(web.Files)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:      cfg,
		assetVer: version,
		log:      log,
		renderer: renderer,
		monitor:  metrics,
		sessions: sessions,
		limiter:  limiter,
		creds:    creds,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	protected := auth.RequireSession(s.sessions)

	mux.HandleFunc("GET /{$}", s.handleRoot)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /signin", s.handleSignInForm)
	mux.HandleFunc("POST /signin", s.handleSignIn)
	mux.HandleFunc("POST /signout", s.handleSignOut)

	mux.Handle("GET /dashboard", protected(http.HandlerFunc(s.handleDashboard)))
	mux.Handle("GET /memory", protected(http.HandlerFunc(s.handleMemory)))
	mux.Handle("GET /memory/live", protected(http.HandlerFunc(s.handleMemoryLive)))
	mux.Handle("GET /system", protected(http.HandlerFunc(s.handleSystem)))
	mux.Handle("GET /docker", protected(http.HandlerFunc(s.handleDocker)))
	mux.Handle("GET /docker/live", protected(http.HandlerFunc(s.handleDockerLive)))
	mux.Handle("GET /api/live-stats", protected(http.HandlerFunc(s.handleLiveStats)))

	mux.Handle("GET /static/", staticHandler(web.Files, s.assetVer))

	return chain(mux,
		recoverPanic(s.log),
		logRequests(s.log, s.cfg.TrustProxyHeader),
		securityHeaders,
		limitBody,
		auth.CSRF(s.cfg.CookieSecure),
	)
}
