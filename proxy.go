package main

// Двухэтапная схема перевода через Yandex Translate (turbopages):
//   1. получаем proxy_u-префикс (заглушка http://);
//   2. к префиксу приклеиваем целевой URL (в формате https/хост/путь),
//      качаем страницу и вырезаем текст.
//
// DNS-резолвинг — системный (как у браузера): никакого явного DNS-сервера.

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// maxBodyBytes — предел размера скачиваемой страницы (10 МБ).
	maxBodyBytes = 10 << 20

	// prefixTTL — сколько живёт закэшированный proxy_u-префикс.
	prefixTTL = 5 * time.Minute

	userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
)

// httpClient использует системный резолвер.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

type prefixEntry struct {
	prefix  string
	lang    string
	expires time.Time
}

type prefixCache struct {
	mu    sync.Mutex
	entry *prefixEntry
}

var cache = &prefixCache{}

// getPrefix возвращает proxy_u-префикс для языка lang (этап 1), с кэшем.
func (c *prefixCache) getPrefix(lang string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entry != nil && c.entry.lang == lang && time.Now().Before(c.entry.expires) {
		return c.entry.prefix, nil
	}

	p, err := fetchPrefix(lang)
	if err != nil {
		return "", err
	}
	c.entry = &prefixEntry{prefix: p, lang: lang, expires: time.Now().Add(prefixTTL)}
	return p, nil
}

func (c *prefixCache) invalidate() {
	c.mu.Lock()
	c.entry = nil
	c.mu.Unlock()
}

// fetchPrefix: заглушка http:// нужна, чтобы Location вернул чистый префикс,
// заканчивающийся на '/'.
func fetchPrefix(lang string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		// не следуем редиректу — нам нужен именно заголовок Location.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	reqURL := fmt.Sprintf("https://translate.yandex.ru/translate?url=http://&lang=%s", url.QueryEscape(lang))
	resp, err := client.Get(reqURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no Location header (status %d)", resp.StatusCode)
	}
	return loc, nil
}

// toProxyPath: обычный URL (https://host/path) -> формат прокси (https/host/path).
func toProxyPath(target string) string {
	return strings.Replace(target, "://", "/", 1)
}

// fetchTranslatedText: этап 2 — склейка префикса и цели, скачивание, текст.
func fetchTranslatedText(target, lang string) (string, error) {
	prefix, err := cache.getPrefix(lang)
	if err != nil {
		return "", err
	}

	text, err := fetchAndExtract(prefix + toProxyPath(target))
	if err == nil {
		return text, nil
	}

	// Префикс мог протухнуть — обновляем и пробуем ещё раз.
	cache.invalidate()
	prefix, err = cache.getPrefix(lang)
	if err != nil {
		return "", err
	}
	return fetchAndExtract(prefix + toProxyPath(target))
}

func fetchAndExtract(fullURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	if !isTextContent(resp.Header.Get("Content-Type")) {
		return "", fmt.Errorf("unsupported content-type %q", resp.Header.Get("Content-Type"))
	}

	// Читаем с лимитом, чтобы не загружать в память огромные ответы.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxBodyBytes {
		return "", fmt.Errorf("response too large (>%d bytes)", maxBodyBytes)
	}

	return extractTranslatedText(string(body)), nil
}

// isTextContent проверяет, что это текстовый контент, а не бинарный.
func isTextContent(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	if strings.HasPrefix(ct, "text/") {
		return true
	}
	switch ct {
	case "application/json", "application/xml", "application/xhtml+xml":
		return true
	}
	return false
}
