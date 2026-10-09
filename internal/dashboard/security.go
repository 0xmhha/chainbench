package dashboard

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
)

const webCookie = "chainbench_session"

// WebAuthenticator adapts server-owned cookie sessions to the existing engine actors.
func WebAuthenticator(auth *app.WebAuth) DeploymentAuthenticator {
	return func(r *http.Request) (app.DeploymentActor, error) {
		c, err := r.Cookie(webCookie)
		if err != nil {
			return app.DeploymentActor{}, err
		}
		s, err := auth.Session(c.Value)
		role := s.User.Role
		if role == "administrator" {
			role = "admin"
		}
		return app.DeploymentActor{ID: s.User.ID, Role: role}, err
	}
}

// WithWebSecurity protects all API and SSE surfaces; legacy read contracts keep their shapes.
func WithWebSecurity(auth *app.WebAuth, deployments *app.DeploymentStore) Option {
	return func(s *Server) {
		deployments.SetAuthorizer(auth.AuthorizeJobActor)
		s.publisherAudit = auth.AuditWeb
		s.publisherRedact = deployments.RedactWeb
		s.authorizeStream = func(r *http.Request) bool {
			c, err := r.Cookie(webCookie)
			if err != nil {
				return false
			}
			_, err = auth.Session(c.Value)
			return err == nil
		}
		s.security = func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if !strings.HasPrefix(path, "/api/") && path != "/events" {
				s.mux.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			response := &secureResponse{ResponseWriter: w, status: http.StatusOK, redact: deployments.RedactWeb, explain: r.Method != http.MethodGet && r.Method != http.MethodHead}
			// This endpoint is generated exclusively from parser/dialect metadata.
			// A schema property named password describes a file-path option,
			// rather than containing credential material. Preserve its schema.
			if r.Method == "GET" && path == "/api/v1/contracts/chain-preset" {
				response.redact = app.RedactWebText
			}
			actor := "unauthenticated"
			// Record server-selected route patterns only: attacker-controlled URLs may contain secrets.
			operation := "web.request"
			defer func() {
				if r.Method != "GET" || response.status >= 400 {
					_ = auth.AuditWeb(actor, operation, response.status)
				}
			}()
			public := path == "/api/v1/bootstrap" || path == "/api/v1/auth/login" || path == "/api/v1/auth/config"
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			if r.Method != "GET" && r.Method != "HEAD" {
				origin := r.Header.Get("Origin")
				if (origin != "" && origin != scheme+"://"+r.Host) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
					http.Error(response, "forbidden", http.StatusForbidden)
					return
				}
			}
			if !public {
				c, err := r.Cookie(webCookie)
				if err != nil {
					http.Error(response, "login required", http.StatusUnauthorized)
					return
				}
				session, err := auth.Session(c.Value)
				if err != nil {
					http.Error(response, "login required", http.StatusUnauthorized)
					return
				}
				actor = session.User.ID
				if strings.HasPrefix(path, "/api/v1/credentials") && session.User.Role == "viewer" {
					http.Error(response, "operator required", http.StatusForbidden)
					return
				}
				if r.Method != "GET" && r.Method != "HEAD" {
					if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(session.CSRFToken)) != 1 {
						http.Error(response, "CSRF token required", http.StatusForbidden)
						return
					}
					readComparison := r.Method == "POST" && path == "/api/v1/history/compare"
					if path != "/api/v1/auth/logout" && !readComparison && session.User.Role == "viewer" {
						http.Error(response, "read-only account", http.StatusForbidden)
						return
					}
				}
				if strings.HasPrefix(path, "/api/v1/users") && session.User.Role != "administrator" {
					http.Error(response, "administrator required", http.StatusForbidden)
					return
				}
				if r.Method == "DELETE" && strings.HasPrefix(path, "/api/v1/runs") && session.User.Role != "administrator" {
					http.Error(response, "administrator required", http.StatusForbidden)
					return
				}
			}
			if r.Method != "GET" && r.Method != "HEAD" {
				if err := auth.AuditWeb(actor, "web.accepted", 0); err != nil {
					http.Error(response, "audit unavailable", http.StatusInternalServerError)
					return
				}
			}
			s.mux.ServeHTTP(response, r)
			operation = r.Pattern
			if operation == "" {
				operation = "web.unmatched"
			}
		}
		setSession := func(w http.ResponseWriter, r *http.Request, session app.WebSession, token string, status int) {
			http.SetCookie(w, &http.Cookie{Name: webCookie, Value: token, Path: "/", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, Expires: session.ExpiresAt})
			deploymentJSON(w, status, session)
		}
		s.mux.HandleFunc("POST /api/v1/bootstrap", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Username   string `json:"username"`
				Password   string `json:"password"`
				SetupToken string `json:"setupToken"`
			}
			if !deploymentDecode(w, r, &in) {
				return
			}
			session, token, err := auth.Bootstrap(in.Username, in.Password, in.SetupToken)
			if err != nil {
				deploymentError(w, err)
				return
			}
			setSession(w, r, session, token, http.StatusCreated)
		})
		s.mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if !deploymentDecode(w, r, &in) {
				return
			}
			session, token, err := auth.Login(in.Username, in.Password)
			if err != nil {
				http.Error(w, "invalid login", http.StatusUnauthorized)
				return
			}
			setSession(w, r, session, token, http.StatusOK)
		})
		s.mux.HandleFunc("GET /api/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
			c, _ := r.Cookie(webCookie)
			session, err := auth.Session(c.Value)
			if err != nil {
				http.Error(w, "login required", http.StatusUnauthorized)
				return
			}
			deploymentJSON(w, http.StatusOK, session)
		})
		s.mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
			c, _ := r.Cookie(webCookie)
			auth.Logout(c.Value)
			http.SetCookie(w, &http.Cookie{Name: webCookie, Path: "/", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: -1})
			w.WriteHeader(204)
		})
		s.mux.HandleFunc("GET /api/v1/auth/config", func(w http.ResponseWriter, r *http.Request) {
			deploymentJSON(w, http.StatusOK, map[string]any{"bootstrapRequired": auth.BootstrapRequired()})
		})
		authenticate := WebAuthenticator(auth)
		s.mux.HandleFunc("GET /api/v1/users", func(w http.ResponseWriter, r *http.Request) {
			deploymentJSON(w, http.StatusOK, map[string]any{"items": auth.Users(), "nextCursor": nil})
		})
		s.mux.HandleFunc("POST /api/v1/users", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Username string `json:"username"`
				Password string `json:"password"`
				Role     string `json:"role"`
			}
			if !deploymentDecode(w, r, &in) {
				return
			}
			a, _ := authenticate(r)
			u, err := auth.CreateUser(a, in.Username, in.Password, in.Role)
			deploymentResponse(w, u, err, http.StatusCreated)
		})
		s.mux.HandleFunc("PATCH /api/v1/users/{userId}", func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Role     string `json:"role"`
				Password string `json:"password"`
				Active   *bool  `json:"active"`
			}
			if !deploymentDecode(w, r, &in) {
				return
			}
			a, _ := authenticate(r)
			u, err := auth.UpdateUser(a, r.PathValue("userId"), in.Role, in.Password, in.Active)
			if err == nil && s.jobs != nil && (!u.Active || u.Role == "viewer") {
				err = s.jobs.RevokeActor(u.ID)
			}
			deploymentResponse(w, u, err, http.StatusOK)
		})
	}
}

// secureResponse sanitizes each serialized event/response and preserves SSE flushing.
type secureResponse struct {
	http.ResponseWriter
	status  int
	redact  func(string) string
	errored bool
	// explain lets a state-changing request see why its input was refused.
	explain bool
}

func (w *secureResponse) WriteHeader(status int) {
	w.status = status
	w.Header().Del("Content-Length")
	if status >= 400 {
		w.Header().Set("Content-Type", "application/json")
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *secureResponse) Write(b []byte) (int, error) {
	if w.status >= 400 {
		if w.errored {
			return len(b), nil
		}
		w.errored = true
		_, err := w.ResponseWriter.Write(webErrorBody(w.status, b, w.redact, w.explain))
		return len(b), err
	}
	_, err := w.ResponseWriter.Write([]byte(w.redact(string(b))))
	return len(b), err
}
func (w *secureResponse) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Explicit provisioned Basic accounts retain compatibility while protecting all
// legacy observation routes and read-only validation from anonymous or viewer writes.
func legacyWebSecurity(s *Server, store *app.DeploymentStore, authenticate DeploymentAuthenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if (!strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/events") || r.URL.Path == "/api/v1/auth/config" {
			s.mux.ServeHTTP(w, r)
			return
		}
		response := &secureResponse{ResponseWriter: w, status: http.StatusOK, redact: store.RedactWeb, explain: r.Method != http.MethodGet && r.Method != http.MethodHead}
		if authenticate == nil {
			http.Error(response, "login required", http.StatusUnauthorized)
			return
		}
		actor, err := authenticate(r)
		if err != nil || actor.ID == "" {
			http.Error(response, "login required", http.StatusUnauthorized)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/credentials") && actor.Role == "viewer" {
			http.Error(response, "operator required", http.StatusForbidden)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			origin := r.Header.Get("Origin")
			readComparison := r.Method == "POST" && r.URL.Path == "/api/v1/history/compare"
			if (actor.Role == "viewer" && !readComparison) || (origin != "" && origin != scheme+"://"+r.Host) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				http.Error(response, "forbidden", http.StatusForbidden)
				return
			}
		}
		s.mux.ServeHTTP(response, r)
	}
}

// webValidationLimit bounds an explained validation message.
const webValidationLimit = 2048

// webErrorBody keeps the generic status text for every failure except request
// validation (400, 409, 422), whose reason the user needs to correct the input.
// That reason is redacted for registered secrets and personal credentials;
// viewers never reach a write route, so only editors read these reasons.
func webErrorBody(status int, original []byte, redact func(string) string, explain bool) []byte {
	body := map[string]any{"code": http.StatusText(status), "message": http.StatusText(status), "requestId": "web-rejected"}
	if explain && (status == http.StatusBadRequest || status == http.StatusConflict || status == http.StatusUnprocessableEntity) {
		var structured map[string]any
		text := strings.TrimSpace(redact(string(original)))
		if json.Unmarshal([]byte(text), &structured) == nil && len(text) <= 4*webValidationLimit {
			for key, value := range structured {
				if key == "message" {
					if message, ok := value.(string); ok && message != "" {
						body["message"] = message
					}
				} else if key != "code" && key != "requestId" {
					body[key] = value
				}
			}
		} else if text != "" && structured == nil {
			if len(text) > webValidationLimit {
				text = text[:webValidationLimit] + " …"
			}
			body["message"] = text
		}
	}
	b, _ := json.Marshal(body)
	return append(b, '\n')
}
