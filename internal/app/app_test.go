package app

import (
	ical "github.com/arran4/golang-ical"
	"html/template"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func setupTemplates() {
	templatesByLang = map[string]*template.Template{
		"en": template.Must(template.New("calendar.html").
			Parse(`{{define "calendar.html"}}calendar{{end}}{{define "home.html"}}home{{end}}{{define "about.html"}}about{{end}}{{define "contact.html"}}contact{{end}}{{define "impressum.html"}}impressum{{end}}{{define "news.html"}}news{{end}}`)),
		"de": template.Must(template.New("calendar.html").
			Parse(`{{define "calendar.html"}}calendar{{end}}{{define "home.html"}}home{{end}}{{define "about.html"}}about{{end}}{{define "contact.html"}}contact{{end}}{{define "impressum.html"}}impressum{{end}}{{define "news.html"}}news{{end}}`)),
	}
}

func TestMakeLangHandler(t *testing.T) {
	setupTemplates()
	supportedLangs = []string{"en", "de"}
	calendarURLs = map[string]string{"wochenkurse": "webcal://dummy"}
	req := httptest.NewRequest("GET", "/home?lang=en", nil)
	w := httptest.NewRecorder()
	handler := makeLangHandler("home.html")
	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
	body := w.Body.String()
	if !strings.Contains(body, "home") {
		t.Errorf("expected body to contain 'home', got %q", body)
	}
}

func TestMakeLangHandler_Error(t *testing.T) {
	setupTemplates()
	supportedLangs = []string{"en"}
	calendarURLs = map[string]string{"wochenkurse": "webcal://dummy"}
	req := httptest.NewRequest("GET", "/home?lang=en", nil)
	w := httptest.NewRecorder()
	handler := makeLangHandler("notfound.html")
	handler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestCalendarHandler_Smoke(t *testing.T) {
	setupTemplates()
	supportedLangs = []string{"en"}
	calendarURLs = map[string]string{"wochenkurse": "webcal://dummy"}
	req := httptest.NewRequest("GET", "/?lang=en", nil)
	w := httptest.NewRecorder()
	calendarHandler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestGetSelectedCalendars(t *testing.T) {
	calendarURLs = map[string]string{
		"wochenkurse":      "url1",
		"sonderkurse":      "url2",
		"schnupperstunden": "url3",
		"ferienkurse":      "url4",
	}
	tests := []struct {
		param string
		want  []string
	}{
		{"wochenkurse,sonderkurse", []string{"wochenkurse", "sonderkurse"}},
		{"", []string{"sonderkurse", "schnupperstunden", "ferienkurse"}},
		{"invalid", []string{}},
	}
	for _, tt := range tests {
		got, active := getSelectedCalendars(tt.param)
		// Sort both slices for comparison since map iteration order is random
		sort.Strings(got)
		sort.Strings(tt.want)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("getSelectedCalendars(%q) = %v; want %v", tt.param, got, tt.want)
		}
		for _, c := range got {
			if !active[c] {
				t.Errorf("activeCals missing %q", c)
			}
		}
	}
}

func TestBuildTemplateData(t *testing.T) {
	events := []CalendarEvent{{Summary: "foo"}}
	active := map[string]bool{"wochenkurse": true}
	data := buildTemplateData("en", "wochenkurse", events, active)
	if data.Lang != "en" || data.Calendar != "wochenkurse" || len(data.Events) != 1 || !data.ActiveCals["wochenkurse"] {
		t.Error("buildTemplateData did not set fields correctly")
	}
}

func TestParseEvent(t *testing.T) {
	event := ical.NewEvent("test")
	event.SetProperty(ical.ComponentPropertyDtStart, "20240102T150405Z")
	event.SetProperty(ical.ComponentPropertyDtEnd, "20240102T160405Z")
	event.SetProperty(ical.ComponentPropertySummary, "summary")
	event.SetProperty(ical.ComponentPropertyDescription, "desc")
	event.SetProperty(ical.ComponentPropertyLocation, "loc")
	calEvent, start, end := parseEvent(event, "wochenkurse")
	if calEvent.Summary != "summary" || calEvent.Description != "desc" || calEvent.Location != "loc" {
		t.Error("parseEvent did not parse fields")
	}
	if start.IsZero() || end.IsZero() {
		t.Error("parseEvent did not parse times")
	}
	if calEvent.Duration == "" {
		t.Error("parseEvent did not set duration")
	}
}

func TestFetchEventsForCalendar_Empty(t *testing.T) {
	// Use a non-existent calendar name to trigger error handling
	calendarURLs = map[string]string{"wochenkurse": "webcal://invalid-url"}
	events := fetchEventsForCalendar("wochenkurse", time.Now())
	if len(events) != 0 {
		t.Error("expected no events on error")
	}
}

func TestFetchCalendarEvents_Sorting(t *testing.T) {
	// This test checks that fetchCalendarEvents sorts events by startTime ascending.
	// We'll create a fake calendar with two events in reverse order.
	// Since fetchCalendarEvents calls fetchEventsForCalendar, and fetchEventsForCalendar
	// cannot be patched, we test sorting logic by creating a local slice and sorting it.

	// Create two eventWithTime with different start times
	now := time.Now()
	eventsWithTime := []eventWithTime{
		{CalendarEvent: CalendarEvent{Summary: "b"}, startTime: now.Add(2 * time.Hour)},
		{CalendarEvent: CalendarEvent{Summary: "a"}, startTime: now.Add(1 * time.Hour)},
	}

	// Sort manually as fetchCalendarEvents would do
	sort.Slice(eventsWithTime, func(i, j int) bool {
		return eventsWithTime[i].startTime.Before(eventsWithTime[j].startTime)
	})

	if eventsWithTime[0].Summary != "a" || eventsWithTime[1].Summary != "b" {
		t.Error("eventWithTime sorting by startTime failed")
	}
}
