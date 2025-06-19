package main

import (
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

var supportedLangs = []string{"en", "de"}
var templatesByLang map[string]*template.Template

// Struct for passing data to templates
type TemplateData struct {
	Page string
	Lang string
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	loadTemplates()
	http.HandleFunc("/", makeLangHandler("home.html"))
	http.HandleFunc("/about", makeLangHandler("about.html"))
	http.HandleFunc("/contact", makeLangHandler("contact.html"))
	http.HandleFunc("/impressum", makeLangHandler("impressum.html"))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("static/images"))))
	slog.Info("Server brutally started at http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		slog.Error("failed to run server", "err", err.Error())
		os.Exit(1)
	}
}

func loadTemplates() {
	templatesByLang = make(map[string]*template.Template)
	for _, lang := range supportedLangs {
		pattern := filepath.Join("static", "templates", lang, "*.html")
		templatesByLang[lang] = template.Must(template.ParseGlob(pattern))
	}
}

func getLang(r *http.Request) string {
	lang := r.URL.Query().Get("lang")
	for _, l := range supportedLangs {
		if lang == l {
			return l
		}
	}
	return "de"
}

func makeLangHandler(page string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := getLang(r)
		tmpl := templatesByLang[lang]
		data := TemplateData{
			Page: page[:len(page)-5], // e.g., "home"
			Lang: lang,
		}
		slog.Debug("renderTemplate", "lang", lang, "page", page)
		err := tmpl.ExecuteTemplate(w, page, data)
		if err != nil {
			slog.Error("failed to render template", "err", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
