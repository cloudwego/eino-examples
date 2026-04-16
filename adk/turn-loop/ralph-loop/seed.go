/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"log"

	"github.com/cloudwego/eino/adk/filesystem"
)

// seedBuggyProject pre-populates the InMemoryBackend with a URL shortener
// project that has multiple intentional bugs and missing features.
// The Ralph Loop agent must iteratively find and fix all issues.
func seedBuggyProject(ctx context.Context, backend *filesystem.InMemoryBackend) {
	files := map[string]string{
		"/project/store.go":        storeGo,
		"/project/handler.go":      handlerGo,
		"/project/handler_test.go": handlerTestGo,
		"/project/main.go":         mainGo,
		"/project/README.md":       readmeMd,
	}
	for path, content := range files {
		if err := backend.Write(ctx, &filesystem.WriteRequest{
			FilePath: path,
			Content:  content,
		}); err != nil {
			log.Fatalf("seed %s: %v", path, err)
		}
	}
}

// --- Buggy starter files ---
// Each file has intentional bugs listed in the task prompt.

const storeGo = `package shortener

import (
	"math/rand"
	"sync"
)

const codeLen = 6
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type URLEntry struct {
	URL  string
	Hits int
}

type URLStore struct {
	mu       sync.RWMutex
	codes    map[string]*URLEntry // code -> entry
	urls     map[string]string    // url -> code (reverse index)
}

func NewURLStore() *URLStore {
	return &URLStore{
		codes: make(map[string]*URLEntry),
		urls:  make(map[string]string),
	}
}

// BUG: Does not validate URLs (scheme + host required).
// BUG: Does not check for duplicate URLs.
func (s *URLStore) Shorten(rawURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	code := generateCode()
	s.codes[code] = &URLEntry{URL: rawURL}
	return code, nil
}

func (s *URLStore) Resolve(code string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.codes[code]
	if !ok {
		return "", fmt.Errorf("code not found")
	}
	return entry.URL, nil
}

// BUG: Stats always returns 0 hits regardless of actual count.
func (s *URLStore) Stats(code string) (string, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.codes[code]
	if !ok {
		return "", 0, fmt.Errorf("code not found")
	}
	return entry.URL, 0, nil // BUG: hardcoded 0 instead of entry.Hits
}

func generateCode() string {
	b := make([]byte, codeLen)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
`

const handlerGo = `package shortener

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Handler struct {
	Store *URLStore
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/shorten":
		h.handleShorten(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/stats/"):
		h.handleStats(w, r)
	case r.Method == http.MethodGet && len(r.URL.Path) > 1:
		h.handleRedirect(w, r)
	default:
		http.NotFound(w, r)
	}
}

// BUG: Returns 200 instead of 201.
func (h *Handler) handleShorten(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string ` + "`json:\"url\"`" + `
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// BUG: Error response is plain text, should be JSON.
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	code, err := h.Store.Shorten(req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// BUG: Should return 201 Created, not 200 OK.
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"short_code": code,
		"short_url":  fmt.Sprintf("http://localhost:8080/%s", code),
	})
}

// BUG: Does not increment the hit counter.
func (h *Handler) handleRedirect(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/")

	url, err := h.Store.Resolve(code)
	if err != nil {
		// BUG: Error response is plain text, should be JSON.
		http.Error(w, "code not found", http.StatusNotFound)
		return
	}

	// BUG: Missing hit counter increment.
	http.Redirect(w, r, url, http.StatusMovedPermanently)
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/stats/")

	url, hits, err := h.Store.Stats(code)
	if err != nil {
		// BUG: Error response is plain text, should be JSON.
		http.Error(w, "code not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"url":  url,
		"hits": hits,
	})
}
`

const handlerTestGo = `package shortener

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// BUG: Only 3 test cases. Needs at least 8.
// BUG: Does not test error responses (400, 404, 409).

func TestShortenValid(t *testing.T) {
	store := NewURLStore()
	h := &Handler{Store: store}

	body := strings.NewReader(` + "`" + `{"url":"https://example.com"}` + "`" + `)
	req := httptest.NewRequest(http.MethodPost, "/shorten", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK { // Note: should check for 201 after fix
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestRedirectValid(t *testing.T) {
	store := NewURLStore()
	code, _ := store.Shorten("https://example.com")
	h := &Handler{Store: store}

	req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301, got %d", w.Code)
	}
}

func TestStatsValid(t *testing.T) {
	store := NewURLStore()
	code, _ := store.Shorten("https://example.com")
	h := &Handler{Store: store}

	req := httptest.NewRequest(http.MethodGet, "/stats/"+code, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
`

const mainGo = `package main

import (
	"fmt"
	"net/http"
)

func main() {
	store := shortener.NewURLStore()
	handler := &shortener.Handler{Store: store}

	fmt.Println("Server starting on :8080")
	// BUG: No graceful shutdown.
	http.ListenAndServe(":8080", handler)
}
`

const readmeMd = "# URL Shortener\n\nA simple URL shortener service.\n\n## API\n\n### POST /shorten\n\nShorten a URL.\n\n```bash\ncurl -X POST http://localhost:8080/shorten -H \"Content-Type: application/json\" -d '{\"url\":\"https://example.com\"}'\n```\n\n### GET /:code\n\nRedirect to the original URL.\n\n### GET /stats/:code\n\nGet stats for a shortened URL.\n\n<!-- BUG: Missing curl examples for error cases -->\n"
