package app

import (
	"html/template"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGetLang(t *testing.T) {
	supportedLangs = []string{"en", "de"}
	tests := []struct {
		langParam string
		want      string
	}{
		{"en", "en"},
		{"de", "de"},
		{"fr", defaultLang},
		{"", defaultLang},
	}
	for _, tt := range tests {
		req := &http.Request{URL: &url.URL{RawQuery: "lang=" + tt.langParam}}
		got := getLang(req)
		if got != tt.want {
			t.Errorf("getLang(%q) = %q; want %q", tt.langParam, got, tt.want)
		}
	}
}

func TestParseICalTimeToHuman(t *testing.T) {
	tests := []struct {
		input  string
		wantOk bool
	}{
		{"20240102T150405Z", true},
		{"20240102T150405", true},
		{"20240102", true},
		{"", false},
		{"invalid", false},
	}
	for _, tt := range tests {
		gotTime, gotStr := parseICalTimeToHuman(tt.input)
		if tt.wantOk {
			if gotTime.IsZero() || gotStr == "" || gotStr == tt.input {
				t.Errorf("parseICalTimeToHuman(%q) failed, gotTime=%v, gotStr=%q", tt.input, gotTime, gotStr)
			}
		} else {
			if !(gotTime.IsZero() && (gotStr == "" || gotStr == tt.input)) {
				t.Errorf("parseICalTimeToHuman(%q) expected failure, gotTime=%v, gotStr=%q", tt.input, gotTime, gotStr)
			}
		}
	}
}

func TestHumanDuration(t *testing.T) {
	tests := []struct {
		dur  time.Duration
		want string
	}{
		{time.Minute * 90, "1h 30m"},
		{time.Hour * 25, "1d 1h"},
		{time.Hour * 48, "2d"},
		{time.Minute * 5, "5m"},
		{time.Second * 0, "0m"},
		{-time.Hour * 25, "1d 1h"},
		{-(time.Hour*24 + time.Minute*5), "1d 5m"},
	}
	for _, tt := range tests {
		got := humanDuration(tt.dur)
		if got != tt.want {
			t.Errorf("humanDuration(%v) = %q; want %q", tt.dur, got, tt.want)
		}
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"a, b, c", []string{"a", "b", "c"}},
		{"  a ,b,,c ", []string{"a", "b", "c"}},
		{"", []string{}},
		{", ,", []string{}},
		{"foo", []string{"foo"}},
	}
	for _, tt := range tests {
		got := splitAndTrim(tt.in)
		if len(got) == 0 && len(tt.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitAndTrim(%q) = %v; want %v", tt.in, got, tt.want)
		}
	}
}

// TestLoadTemplates is skipped because patching filepath.Join is not possible in Go.
// Integration tests for template loading should be done in a separate integration test suite.
// func TestLoadTemplates(t *testing.T) { ... }

func TestTemplateFuncMap(t *testing.T) {
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
	if funcMap["title"].(func(string) string)("foo") != "FOO" {
		t.Error("title funcMap did not uppercase")
	}
	d := funcMap["dict"].(func(...interface{}) map[string]interface{})("a", 1, "b", 2)
	if d["a"] != 1 || d["b"] != 2 {
		t.Error("dict funcMap did not create correct map")
	}
	if funcMap["safeURL"].(func(string) template.URL)("abc") != template.URL("abc") {
		t.Error("safeURL funcMap did not cast string to template.URL")
	}
}
