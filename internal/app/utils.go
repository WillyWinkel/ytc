package app

import (
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
)

const defaultLang = "de"

//go:embed static/templates/*/*.html
var templatesFS embed.FS

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
		pattern := "static/templates/" + lang + "/*.html"
		files, err := fs.Glob(templatesFS, pattern)
		if err != nil {
			slog.Error("failed to glob templates", "lang", lang, "err", err)
			continue
		}
		tmpl := template.New("").Funcs(funcMap)
		if len(files) == 0 {
			slog.Error("no templates found", "lang", lang)
			continue
		}
		if _, err := tmpl.ParseFS(templatesFS, files...); err != nil {
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
