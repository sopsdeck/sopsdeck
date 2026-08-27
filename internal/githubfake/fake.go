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
	http    *httptest.Server
}

func New() *Server {
	s := &Server{secrets: map[string]string{}}
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

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/actions/secrets/public-key"):
		_ = json.NewEncoder(w).Encode(map[string]string{
			"key_id": "studio",
			"key":    "dGVzdA==",
		})
	case r.Method == http.MethodGet && strings.Contains(path, "/actions/secrets"):
		s.mu.Lock()
		var list []map[string]string
		for name := range s.secrets {
			list = append(list, map[string]string{"name": name})
		}
		s.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"secrets": list})
	case r.Method == http.MethodPut && strings.Contains(path, "/actions/secrets/"):
		name := path[strings.LastIndex(path, "/")+1:]
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.secrets[name] = string(body)
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	case r.Method == http.MethodDelete && strings.Contains(path, "/actions/secrets/"):
		name := path[strings.LastIndex(path, "/")+1:]
		s.mu.Lock()
		delete(s.secrets, name)
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}
