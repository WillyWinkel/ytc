package app

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const defaultLang = "de"

// loadTemplates parses templates for all supported languages and stores them in templatesByLang.
func loadTemplates() {
	templatesByLang = make(map[string]*template.Template)
	funcMap := template.FuncMap{
		"title": func(s string) string { return strings.ToTitle(s) },
		"dict": func(values ...interface{}) map[string]interface{} {
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values)-1; i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				dict[key] = values[i+1]
			}
			return dict
		},
		"safeURL": func(u string) template.URL { return template.URL(u) },
	}
	for _, lang := range supportedLangs {
		pattern := filepath.Join("static", "templates", lang, "*.html")
		tmpl, err := template.New("").Funcs(funcMap).ParseGlob(pattern)
		if err != nil {
			slog.Error("failed to parse templates", "lang", lang, "err", err)
			continue
		}
		templatesByLang[lang] = tmpl
	}
}

// getLang returns the requested language if supported, otherwise the default.
func getLang(r *http.Request) string {
	lang := r.URL.Query().Get("lang")
	for _, l := range supportedLangs {
		if lang == l {
			return l
		}
	}
	return defaultLang
}

// parseICalTimeToHuman parses an iCal time string and returns the parsed time and a human-readable string.
func parseICalTimeToHuman(value string) (time.Time, string) {
	if value == "" {
		slog.Error("parseICalTimeToHuman: empty value")
		return time.Time{}, ""
	}
	layouts := []struct {
		layout string
		format string
	}{
		{"20060102T150405Z", "2.1.2006 15:04"},
		{"20060102T150405", "2.1.2006 15:04"},
		{"20060102", "2.1.2006"},
	}
	for _, l := range layouts {
		if t, err := time.Parse(l.layout, value); err == nil {
			return t, t.Format(l.format)
		}
	}
	slog.Error("parseICalTimeToHuman: failed to parse", "value", value)
	return time.Time{}, value
}

// humanDuration returns a human-readable duration string for a time.Duration.
func humanDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	m := int(d.Minutes()) % 60
	switch {
	case days > 0 && h > 0 && m > 0:
		return fmt.Sprintf("%dd %dh %dm", days, h, m)
	case days > 0 && h > 0:
		return fmt.Sprintf("%dd %dh", days, h)
	case days > 0 && m > 0:
		return fmt.Sprintf("%dd %dm", days, m)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	case m > 0:
		return fmt.Sprintf("%dm", m)
	}
	return "0m"
}

// splitAndTrim splits a comma-separated string and trims whitespace from each part, omitting empty results.
func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
