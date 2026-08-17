package main

// textproxy — HTTP-сервис: по переданному URL отдаёт чистый текст страницы.
//
// Использование:
//
//	GET /?url=https://example.com/&lang=de-de
//	GET /healthz
//
// Порт задаётся флагом -port или переменной окружения PORT (по умолчанию 8080).

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const defaultLang = "de-de"

func main() {
	port := flag.String("port", getenv("PORT", "8080"), "порт для HTTP-сервера")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/", handleFetch)

	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("textproxy слушает :%s", *port)
	log.Fatal(srv.ListenAndServe())
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "ok\n")
}

func handleFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	target := r.URL.Query().Get("url")
	if target == "" {
		http.Error(w, "missing 'url' query parameter", http.StatusBadRequest)
		return
	}

	u, err := url.Parse(target)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "'url' must be an absolute http(s) URL", http.StatusBadRequest)
		return
	}

	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = defaultLang
	}

	text, err := fetchTranslatedText(target, lang)
	if err != nil {
		http.Error(w, fmt.Sprintf("fetch failed: %v", err), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, text)
}
