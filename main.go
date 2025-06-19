package main

import (
	ical "github.com/arran4/golang-ical"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

var supportedLangs = []string{"en", "de"}
var templatesByLang map[string]*template.Template

// Struct for passing data to templates
type CalendarEvent struct {
	Summary     string
	Description string
	Start       string
	End         string
	Location    string
	Duration    string
}

type TemplateData struct {
	Page   string
	Lang   string
	Events []CalendarEvent
}

const calendarURL = "https://your-ics-url" // Replace with your actual ICS URL

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	loadTemplates()
	http.HandleFunc("/", homeHandler)
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

// Home handler: fetch calendar, parse events, pass to template
func homeHandler(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	tmpl := templatesByLang[lang]

	file, err := os.Open("static/demo.ics")
	if err != nil {
		slog.Error("failed to open calendar file", "err", err.Error())
		http.Error(w, "Failed to load calendar file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	cal, err := ical.ParseCalendar(file)
	// cal, err := ical.ParseCalendarFromUrl(calendarURL)
	if err != nil {
		slog.Error("failed to parse calendar", "err", err.Error())
		http.Error(w, "Failed to parse calendar", http.StatusInternalServerError)
		return
	}

	var events []CalendarEvent
	for _, e := range cal.Events() {
		events = append(events, CalendarEvent{
			Summary:     e.GetProperty(ical.ComponentPropertySummary).Value,
			Description: e.GetProperty(ical.ComponentPropertyDescription).Value,
			Start:       e.GetProperty(ical.ComponentPropertyDtStart).Value,
			End:         e.GetProperty(ical.ComponentPropertyDtEnd).Value,
			Location:    e.GetProperty(ical.ComponentPropertyLocation).Value,
			Duration:    e.GetProperty(ical.ComponentPropertyDuration).Value,
		})
	}

	data := TemplateData{
		Page:   "home",
		Lang:   lang,
		Events: events,
	}
	slog.Debug("renderTemplate", "lang", lang, "page", "home.html", "events", len(events))
	err = tmpl.ExecuteTemplate(w, "home.html", data)
	if err != nil {
		slog.Error("failed to render template", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
