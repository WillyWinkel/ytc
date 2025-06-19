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
	"strings"
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
	Calendar    string // Add calendar name for color/dot
}

type TemplateData struct {
	Page          string
	Lang          string
	Events        []CalendarEvent
	Calendar      string
	Calendars     []string
	CalColors     map[string]string
	ActiveCals    map[string]bool
	CalBtnClasses map[string]string
	CalWebcalURLs map[string]string
}

// Map calendar names to their ICS URLs
var calendarURLs = map[string]string{
	"wochenkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfxreWnKQdW0FFtX6payfjYjJTJFZe4xHvR0bHx3C2wBYAq2682Ughg9wGEjVii8uEs",
	"sonderkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfwnZeAR3LQOhWWLb268k4gqa1jhmgoL-XsvLo6wcVXyHeG_di75FEtbP2difn6tV9Y",
	"schnupperstunden": "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfzT5ZB2ZS9ej1khBvIrOwaOx_Yvn3-WSwh8yMj25fiiKNXTMWQ-y4HQBcjnTGJClXc",
	"ferienkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfw0uWa7nlulHIUfnj6U_loZyYiyTZZaOUxNS2s5lrWQCZTmfIe5Zl__8qw2ZWC1-g0",
}

// Assign a color to each calendar
var calendarColors = map[string]string{
	"wochenkurse":      "#0d6efd", // blue
	"sonderkurse":      "#198754", // green
	"schnupperstunden": "#ffc107", // yellow
	"ferienkurse":      "#dc3545", // red
}

// Assign a Bootstrap btn color class to each calendar
var calendarBtnClasses = map[string]string{
	"wochenkurse":      "primary",
	"sonderkurse":      "success",
	"schnupperstunden": "warning",
	"ferienkurse":      "danger",
}

// Assign a webcal URL (no protocol) to each calendar for download
var calendarWebcalURLs = map[string]string{
	"wochenkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfxreWnKQdW0FFtX6payfjYjJTJFZe4xHvR0bHx3C2wBYAq2682Ughg9wGEjVii8uEs",
	"sonderkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfwnZeAR3LQOhWWLb268k4gqa1jhmgoL-XsvLo6wcVXyHeG_di75FEtbP2difn6tV9Y",
	"schnupperstunden": "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfzT5ZB2ZS9ej1khBvIrOwaOx_Yvn3-WSwh8yMj25fiiKNXTMWQ-y4HQBcjnTGJClXc",
	"ferienkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfw0uWa7nlulHIUfnj6U_loZyYiyTZZaOUxNS2s5lrWQCZTmfIe5Zl__8qw2ZWC1-g0",
}

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
	funcMap := template.FuncMap{
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return string([]rune(s)[0]-32) + s[1:]
		},
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
		"safeURL": func(u string) template.URL { // <-- Add this function
			return template.URL(u)
		},
	}
	for _, lang := range supportedLangs {
		pattern := filepath.Join("static", "templates", lang, "*.html")
		templatesByLang[lang] = template.Must(template.New("").Funcs(funcMap).ParseGlob(pattern))
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

// Formats duration as "Xd Yh Zm"
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

// Home handler: fetch calendar, parse events, pass to template
func homeHandler(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	tmpl := templatesByLang[lang]

	// Parse selected calendars from query
	calendarParam := r.URL.Query().Get("calendar")
	var selectedCalendars []string
	activeCals := make(map[string]bool)
	if calendarParam != "" {
		for _, c := range splitAndTrim(calendarParam) {
			if _, ok := calendarURLs[c]; ok {
				selectedCalendars = append(selectedCalendars, c)
				activeCals[c] = true
			}
		}
	} else {
		// If nothing is selected, select all calendars
		for cal := range calendarURLs {
			selectedCalendars = append(selectedCalendars, cal)
			activeCals[cal] = true
		}
	}
	// No default selection if nothing is selected

	type eventWithTime struct {
		CalendarEvent
		startTime time.Time
		endTime   time.Time
	}

	var eventsWithTime []eventWithTime
	now := time.Now()

	for _, calName := range selectedCalendars {
		calendarURL := calendarURLs[calName]
		calendarURL = strings.Replace(calendarURL, "webcal://", "https://", -1)
		cal, err := ical.ParseCalendarFromUrl(calendarURL)
		if err != nil {
			slog.Error("failed to parse calendar", "calendar", calName, "err", err.Error())
			continue
		}
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
						Calendar:    calName,
					},
					startTime: startTime,
					endTime:   endTime,
				})
			}
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
		Page:          "home",
		Lang:          lang,
		Events:        events,
		Calendar:      calendarParam,
		Calendars:     []string{"wochenkurse", "sonderkurse", "schnupperstunden", "ferienkurse"},
		CalColors:     calendarColors,
		ActiveCals:    activeCals,
		CalBtnClasses: calendarBtnClasses,
		CalWebcalURLs: calendarWebcalURLs,
	}
	slog.Debug("renderTemplate", "lang", lang, "page", "home.html", "events", len(events))
	err := tmpl.ExecuteTemplate(w, "home.html", data)
	if err != nil {
		slog.Error("failed to render template", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Helper: split comma-separated and trim
func splitAndTrim(s string) []string {
	var out []string
	for _, part := range splitComma(s) {
		p := trimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
func splitComma(s string) []string {
	var out []string
	start := 0
	for i, c := range s {
		if c == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n') {
		j--
	}
	return s[i:j]
}

func makeLangHandler(page string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := getLang(r)
		tmpl := templatesByLang[lang]
		data := TemplateData{
			Page:          page[:len(page)-5], // e.g., "home"
			Lang:          lang,
			CalWebcalURLs: calendarWebcalURLs, // <-- ensure this is set for all pages
		}
		slog.Debug("renderTemplate", "lang", lang, "page", page)
		err := tmpl.ExecuteTemplate(w, page, data)
		if err != nil {
			slog.Error("failed to render template", "err", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
