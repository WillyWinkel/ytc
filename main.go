package main

import (
	"fmt"
	ical "github.com/arran4/golang-ical"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
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

const calendarURL = "https://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfxreWnKQdW0FFtX6payfjYjJTJFZe4xHvR0bHx3C2wBYAq2682Ughg9wGEjVii8uEs" // Replace with your actual ICS URL

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

// Parses iCal datetime (e.g., 20240609T090000Z or 20240609) and returns time.Time and human-readable string
func parseICalTimeToHuman(value string) (time.Time, string) {
	if value == "" {
		slog.Error("failed to parse ICal Time to Human", "value", value)
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
		t, err := time.Parse(l.layout, value)
		if err == nil {
			return t, t.Format(l.format)
		}
	}
	slog.Error("failed to parse ICal Time to Human. returning fallback", "value", value)
	return time.Time{}, value // fallback to raw if parsing fails
}

// Formats duration as "Xh Ym"
func humanDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	} else if h > 0 {
		return fmt.Sprintf("%dh", h)
	} else if m > 0 {
		return fmt.Sprintf("%dm", m)
	}
	return "0m"
}

// Home handler: fetch calendar, parse events, pass to template
func homeHandler(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	tmpl := templatesByLang[lang]

	//file, err := os.Open("static/demo.ics")
	//if err != nil {
	//	slog.Error("failed to open calendar file", "err", err.Error())
	//	http.Error(w, "Failed to load calendar file", http.StatusInternalServerError)
	//	return
	//}
	//defer file.Close()

	// cal, err := ical.ParseCalendar(file)
	cal, err := ical.ParseCalendarFromUrl(calendarURL)
	if err != nil {
		slog.Error("failed to parse calendar", "err", err.Error())
		http.Error(w, "Failed to parse calendar", http.StatusInternalServerError)
		return
	}

	type eventWithTime struct {
		CalendarEvent
		startTime time.Time
		endTime   time.Time
	}

	var eventsWithTime []eventWithTime
	now := time.Now()

	for _, e := range cal.Events() {
		var startStr, endStr, summary, description, location string
		var startTime, endTime time.Time

		if prop := e.GetProperty(ical.ComponentPropertyDtStart); prop != nil {
			startTime, startStr = parseICalTimeToHuman(prop.Value)
		}
		if prop := e.GetProperty(ical.ComponentPropertyDtEnd); prop != nil {
			endTime, endStr = parseICalTimeToHuman(prop.Value)
		}
		if prop := e.GetProperty(ical.ComponentPropertySummary); prop != nil {
			summary = prop.Value
		}
		if prop := e.GetProperty(ical.ComponentPropertyDescription); prop != nil {
			description = prop.Value
		}
		if prop := e.GetProperty(ical.ComponentPropertyLocation); prop != nil {
			location = prop.Value
		}

		duration := ""
		if !startTime.IsZero() && !endTime.IsZero() {
			duration = humanDuration(endTime.Sub(startTime))
		}

		// Only add if event is not in the past (endTime >= now)
		if !endTime.IsZero() && endTime.After(now) {
			eventsWithTime = append(eventsWithTime, eventWithTime{
				CalendarEvent: CalendarEvent{
					Summary:     summary,
					Description: description,
					Start:       startStr,
					End:         endStr,
					Location:    location,
					Duration:    duration,
				},
				startTime: startTime,
				endTime:   endTime,
			})
		}
	}

	// Sort by startTime ascending
	sort.Slice(eventsWithTime, func(i, j int) bool {
		return eventsWithTime[i].startTime.Before(eventsWithTime[j].startTime)
	})

	// Extract CalendarEvent slice
	events := make([]CalendarEvent, 0, len(eventsWithTime))
	for _, e := range eventsWithTime {
		events = append(events, e.CalendarEvent)
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
