package httpx

import (
	"net/http"

	"github.com/syamxm/debian-watch/internal/auth"
)

type pageData struct {
	Title     string
	Active    string
	ShowNav   bool
	CSRFToken string
	AssetVer  string
	Data      any
}

type signInView struct {
	Error string
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if auth.HasSession(r, s.sessions) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleSignInForm(w http.ResponseWriter, r *http.Request) {
	if auth.HasSession(r, s.sessions) {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	s.renderSignIn(w, r, http.StatusOK, "")
}

func (s *Server) handleSignIn(w http.ResponseWriter, r *http.Request) {
	ip := ClientIP(r, s.cfg.TrustProxyHeader)
	if !s.limiter.Allowed(ip) {
		s.log.Warn("login rate limited", "ip", ip)
		s.renderSignIn(w, r, http.StatusTooManyRequests, "Too many failed attempts. Try again later.")
		return
	}

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")
	if !s.creds.Verify(username, password) {
		s.limiter.RecordFailure(ip)
		s.log.Warn("failed login", "ip", ip)
		s.renderSignIn(w, r, http.StatusUnauthorized, "Invalid username or password.")
		return
	}

	sessionID, err := s.sessions.Create()
	if err != nil {
		s.fail(w, r, "create session", err)
		return
	}
	if err := auth.RotateCSRFToken(w, s.cfg.CookieSecure); err != nil {
		s.fail(w, r, "rotate csrf token", err)
		return
	}

	s.limiter.Reset(ip)
	auth.SetSessionCookie(w, sessionID, s.cfg.SessionTTL, s.cfg.CookieSecure)
	s.log.Info("login succeeded", "ip", ip)
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) handleSignOut(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil {
		s.sessions.Destroy(cookie.Value)
	}
	auth.ClearSessionCookie(w, s.cfg.CookieSecure)
	http.Redirect(w, r, "/signin", http.StatusSeeOther)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "dashboard", "performance overview", s.monitor.View())
}

func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "memory", "memory detail", s.monitor.View())
}

func (s *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "system", "host environment", s.monitor.View())
}

func (s *Server) handleDocker(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "docker", "containers", s.monitor.View())
}

func (s *Server) handleLiveStats(w http.ResponseWriter, r *http.Request) {
	s.renderBlock(w, r, "dashboard", "live-stats", s.monitor.View())
}

func (s *Server) handleMemoryLive(w http.ResponseWriter, r *http.Request) {
	s.renderBlock(w, r, "memory", "memory-detail", s.monitor.View())
}

func (s *Server) handleDockerLive(w http.ResponseWriter, r *http.Request) {
	s.renderBlock(w, r, "docker", "docker-detail", s.monitor.View())
}

func (s *Server) renderPage(w http.ResponseWriter, r *http.Request, page, title string, data any) {
	view := pageData{
		Title:     title,
		Active:    page,
		ShowNav:   true,
		CSRFToken: auth.CSRFToken(r.Context()),
		AssetVer:  s.assetVer,
		Data:      data,
	}

	var err error
	if r.Header.Get("HX-Request") == "true" {
		err = s.renderer.Block(w, http.StatusOK, page, "content", view)
	} else {
		err = s.renderer.Page(w, http.StatusOK, page, view)
	}
	if err != nil {
		s.fail(w, r, "render page", err)
	}
}

func (s *Server) renderBlock(w http.ResponseWriter, r *http.Request, page, block string, data any) {
	if err := s.renderer.Block(w, http.StatusOK, page, block, data); err != nil {
		s.fail(w, r, "render block", err)
	}
}

func (s *Server) renderSignIn(w http.ResponseWriter, r *http.Request, status int, message string) {
	view := pageData{
		Title:     "sign in",
		Active:    "signin",
		CSRFToken: auth.CSRFToken(r.Context()),
		AssetVer:  s.assetVer,
		Data:      signInView{Error: message},
	}
	if err := s.renderer.Page(w, status, "signin", view); err != nil {
		s.fail(w, r, "render sign-in", err)
	}
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, operation string, err error) {
	s.log.Error("request failed", "operation", operation, "path", r.URL.Path, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
