package githubfake

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// Server is an in-process GitHub Actions secrets API. It never stores
// decryptable values in a way tests can read back — list/get return names only.
type Server struct {
	mu      sync.Mutex
	secrets map[string]string
	envs    map[string]map[string]string
	auth    string
	http    *httptest.Server
}

func New() *Server {
	s := &Server{secrets: map[string]string{}, envs: map[string]map[string]string{}}
	s.http = httptest.NewServer(http.HandlerFunc(s.serve))
	return s
}

func (s *Server) URL() string { return s.http.URL }

func (s *Server) Close() { s.http.Close() }

func (s *Server) Names() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.secrets))
	for name := range s.secrets {
		out = append(out, name)
	}
	return out
}

func (s *Server) LastAuthorization() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.auth
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.auth = r.Header.Get("Authorization")
	s.mu.Unlock()
	path := r.URL.Path
	if r.Method == http.MethodGet && strings.HasSuffix(path, "/actions/secrets/public-key") {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"key_id": "studio",
			"key":    "dGVzdA==",
		})
		return
	}
	kind, env, name, ok := parseGitHubSecretsPath(path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if name != "" {
			http.NotFound(w, r)
			return
		}
		s.mu.Lock()
		var list []map[string]string
		if kind == "env" {
			for n := range s.envs[env] {
				list = append(list, map[string]string{"name": n})
			}
		} else {
			for n := range s.secrets {
				list = append(list, map[string]string{"name": n})
			}
		}
		s.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"secrets": list})
	case http.MethodPut:
		if name == "" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		if kind == "env" {
			if s.envs[env] == nil {
				s.envs[env] = map[string]string{}
			}
			s.envs[env][name] = string(body)
		} else {
			s.secrets[name] = string(body)
		}
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		if name == "" {
			http.NotFound(w, r)
			return
		}
		s.mu.Lock()
		if kind == "env" {
			delete(s.envs[env], name)
		} else {
			delete(s.secrets, name)
		}
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func parseGitHubSecretsPath(path string) (kind, env, name string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "repos" {
		return "", "", "", false
	}
	switch parts[3] {
	case "actions":
		if parts[4] != "secrets" {
			return "", "", "", false
		}
		switch len(parts) {
		case 5:
			return "repo", "", "", true
		case 6:
			return "repo", "", parts[5], true
		}
	case "environments":
		if len(parts) < 6 || parts[5] != "secrets" {
			return "", "", "", false
		}
		switch len(parts) {
		case 6:
			return "env", parts[4], "", true
		case 7:
			return "env", parts[4], parts[6], true
		}
	}
	return "", "", "", false
}
