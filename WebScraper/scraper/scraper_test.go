package scraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testPageTitle   = "Test Page"
	errOneResultFmt = "expected 1 result, got %d"
)

// ---------- Тесты extractTitle (парсинг HTML) ----------

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name    string
		html    string
		want    string
		wantErr bool
	}{
		{
			name: "simple_title",
			html: `<html><head><title>Hello World</title></head><body></body></html>`,
			want: "Hello World",
		},
		{
			name: "title_with_whitespace",
			html: `<html><head><title>  Spaced Title  </title></head></html>`,
			want: "Spaced Title",
		},
		{
			name: "empty_title",
			html: `<html><head><title></title></head></html>`,
			want: "",
		},
		{
			name:    "no_title_tag",
			html:    `<html><head></head><body><p>No title here</p></body></html>`,
			wantErr: true,
		},
		{
			name: "title_after_meta",
			html: `<html><head><meta charset="utf-8"><title>After Meta</title></head></html>`,
			want: "After Meta",
		},
		{
			name:    "empty_document",
			html:    ``,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractTitle(strings.NewReader(tc.html))
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (title=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("title = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------- Тесты Run (интеграция с HTTP) ----------

// newTestServer создаёт httptest-сервер, отдающий HTML с заданным <title>.
func newTestServer(title string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><head><title>%s</title></head><body></body></html>", title)
	}))
}

// newSlowServer отвечает с задержкой, превышающей ожидаемый таймаут.
func newSlowServer(delay time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		fmt.Fprint(w, "<html><head><title>Slow</title></head></html>")
	}))
}

func TestRunSingleURL(t *testing.T) {
	srv := newTestServer(testPageTitle)
	defer srv.Close()

	results := Run([]string{srv.URL}, DefaultConfig())

	if len(results) != 1 {
		t.Fatalf(errOneResultFmt, len(results))
	}
	r := results[0]
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.Title != testPageTitle {
		t.Errorf("title = %q, want %q", r.Title, testPageTitle)
	}
}

func TestRunMultipleURLs(t *testing.T) {
	titles := []string{"Alpha", "Beta", "Gamma", "Delta"}
	var urls []string
	var servers []*httptest.Server

	for _, title := range titles {
		srv := newTestServer(title)
		servers = append(servers, srv)
		urls = append(urls, srv.URL)
	}
	defer func() {
		for _, s := range servers {
			s.Close()
		}
	}()

	results := Run(urls, Config{MaxWorkers: 2, Timeout: 5 * time.Second})

	if len(results) != len(titles) {
		t.Fatalf("expected %d results, got %d", len(titles), len(results))
	}

	// Собираем полученные заголовки в set для проверки (порядок не гарантирован).
	got := make(map[string]bool)
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("error for %s: %v", r.URL, r.Err)
			continue
		}
		got[r.Title] = true
	}

	for _, title := range titles {
		if !got[title] {
			t.Errorf("missing title %q in results", title)
		}
	}
}

func TestRunTimeout(t *testing.T) {
	srv := newSlowServer(3 * time.Second)
	defer srv.Close()

	// Таймаут 500ms — сервер отвечает через 3s => ошибка.
	results := Run([]string{srv.URL}, Config{MaxWorkers: 1, Timeout: 500 * time.Millisecond})

	if len(results) != 1 {
		t.Fatalf(errOneResultFmt, len(results))
	}
	if results[0].Err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
}

func TestRunInvalidURL(t *testing.T) {
	results := Run([]string{"http://localhost:1"}, Config{MaxWorkers: 1, Timeout: 2 * time.Second})

	if len(results) != 1 {
		t.Fatalf(errOneResultFmt, len(results))
	}
	if results[0].Err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestRunConcurrencyLimit(t *testing.T) {
	// Запускаем 10 URL через семафор с 2 воркерами — все должны завершиться.
	var urls []string
	var servers []*httptest.Server

	for i := 0; i < 10; i++ {
		srv := newTestServer(fmt.Sprintf("Page %d", i))
		servers = append(servers, srv)
		urls = append(urls, srv.URL)
	}
	defer func() {
		for _, s := range servers {
			s.Close()
		}
	}()

	results := Run(urls, Config{MaxWorkers: 2, Timeout: 5 * time.Second})

	if len(results) != 10 {
		t.Fatalf("expected 10 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("error for %s: %v", r.URL, r.Err)
		}
	}
}
