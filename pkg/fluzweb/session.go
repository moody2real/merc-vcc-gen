package fluzweb

import (
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
)

type Session struct {
	mu      sync.RWMutex
	cookies map[string]string
	path    string
}

func NewSession(cookieHeader, path string) *Session {
	s := &Session{cookies: map[string]string{}, path: path}
	s.merge(parseCookieHeader(cookieHeader))
	return s
}

func LoadSession(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewSession(strings.TrimSpace(string(data)), path), nil
}

func parseCookieHeader(h string) map[string]string {
	m := map[string]string{}
	for _, part := range strings.Split(h, ";") {
		part = strings.TrimSpace(part)
		if i := strings.IndexByte(part, '='); i > 0 {
			m[part[:i]] = part[i+1:]
		}
	}
	return m
}

func (s *Session) Header() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.cookies))
	for k := range s.cookies {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(s.cookies[k])
	}
	return b.String()
}

func (s *Session) merge(m map[string]string) {
	s.mu.Lock()
	for k, v := range m {
		if v != "" {
			s.cookies[k] = v
		}
	}
	s.mu.Unlock()
}

func (s *Session) applySetCookies(cs []*http.Cookie) bool {
	changed := false
	s.mu.Lock()
	for _, c := range cs {
		if c.Value != "" && s.cookies[c.Name] != c.Value {
			s.cookies[c.Name] = c.Value
			changed = true
		}
	}
	s.mu.Unlock()
	return changed
}

func (s *Session) Save() error {
	if s.path == "" {
		return nil
	}
	return os.WriteFile(s.path, []byte(s.Header()), 0o600)
}
