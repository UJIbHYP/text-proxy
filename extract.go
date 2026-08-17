package main

// Извлечение читаемого текста из HTML:
//   - целиком выкидываем script/style/noscript/template;
//   - блочные теги заменяем переносом строки;
//   - снимаем остальные теги;
//   - раскодируем HTML-сущности (&amp; -> & и т.п.).

import (
	"html"
	"regexp"
	"strings"
)

var (
	// Блоки, чьё содержимое — не текст страницы, удаляем целиком.
	// (без backreferences: Go-шный regexp — RE2, он их не поддерживает)
	reDropScript   = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script\s*>`)
	reDropStyle    = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style\s*>`)
	reDropNoScript = regexp.MustCompile(`(?is)<noscript\b[^>]*>.*?</noscript\s*>`)
	reDropTemplate = regexp.MustCompile(`(?is)<template\b[^>]*>.*?</template\s*>`)

	// Блочные теги -> перенос строки, чтобы текст не слипался.
	reBlock = regexp.MustCompile(`(?i)</?(p|div|br|li|tr|pre|h[1-6]|ul|ol|table|section|article|header|footer|blockquote)\b[^>]*>`)

	// Любой оставшийся тег.
	reTag = regexp.MustCompile(`(?s)<[^>]*>`)

	// Хвостовые пробелы/табы в конце строки.
	reTrail = regexp.MustCompile(`(?m)[ \t]+$`)

	// Три и более подряд идущих перевода строки.
	reNL = regexp.MustCompile(`\n{3,}`)
)

// extractText превращает произвольный HTML-фрагмент в читаемый текст.
func extractText(src string) string {
	for _, re := range []*regexp.Regexp{reDropScript, reDropStyle, reDropNoScript, reDropTemplate} {
		src = re.ReplaceAllString(src, "")
	}

	src = reBlock.ReplaceAllString(src, "\n")
	src = reTag.ReplaceAllString(src, "")
	src = html.UnescapeString(src)

	// Нормализуем переводы строк (CRLF/CR -> LF).
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")

	// Убираем пробелы в конце строк и схлопываем пустые строки.
	src = reTrail.ReplaceAllString(src, "")
	src = reNL.ReplaceAllString(src, "\n\n")

	return strings.TrimSpace(src) + "\n"
}

// extractTranslatedText вырезает содержимое переведённой страницы:
// от конца переводной панели (id="closeHeader") до попапа (div.tr-popup),
// затем снимает теги. Если маркеры не найдены — чистим всё как есть.
func extractTranslatedText(src string) string {
	start := strings.Index(src, `id="closeHeader"`)
	if start == -1 {
		return extractText(src)
	}
	// пропускаем хвост кнопки closeHeader (до конца её тега)
	if i := strings.Index(src[start:], ">"); i != -1 {
		start += i + 1
	} else {
		return extractText(src)
	}

	end := strings.Index(src, `<div class="tr-popup"`)
	if end == -1 || end < start {
		return extractText(src)
	}

	return extractText(src[start:end])
}
