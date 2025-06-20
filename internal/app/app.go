package app

import (
	"embed"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/WillyWinkel/ytc/internal/utils"
)

// Embed static files
//
//go:embed static/images/*
var imagesFS embed.FS

//go:embed static/downloads/*
var downloadsFS embed.FS

var supportedLangs = []string{"en", "de"}
var templatesByLang map[string]*template.Template

var calendarURLs = map[string]string{
	"wochenkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfxreWnKQdW0FFtX6payfjYjJTJFZe4xHvR0bHx3C2wBYAq2682Ughg9wGEjVii8uEs",
	"sonderkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfwnZeAR3LQOhWWLb268k4gqa1jhmgoL-XsvLo6wcVXyHeG_di75FEtbP2difn6tV9Y",
	"schnupperstunden": "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfzT5ZB2ZS9ej1khBvIrOwaOx_Yvn3-WSwh8yMj25fiiKNXTMWQ-y4HQBcjnTGJClXc",
	"ferienkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfw0uWa7nlulHIUfnj6U_loZyYiyTZZaOUxNS2s5lrWQCZTmfIe5Zl__8qw2ZWC1-g0",
}

var newsURLs = map[string]string{
	"news": "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfymY060CQ5jlmHwPXxtPa5_JOMNfAPXj82_RGF37kIDBcpYXjSkbDii8EnPXk_IVgY",
}

var calendarColors = map[string]string{
	"wochenkurse":      "#0d6efd",
	"sonderkurse":      "#198754",
	"schnupperstunden": "#ffc107",
	"ferienkurse":      "#dc3545",
}

var calendarBtnClasses = map[string]string{
	"wochenkurse":      "primary",
	"sonderkurse":      "success",
	"schnupperstunden": "warning",
	"ferienkurse":      "danger",
}

type CalendarEvent struct {
	Summary     string
	Description string
	Start       string
	End         string
	Location    string
	Duration    string
	Calendar    string
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

type eventWithTime struct {
	CalendarEvent
	startTime time.Time
	endTime   time.Time
}

type DownloadFile struct {
	Name        string
	URL         string
	Description string
}

type DownloadTemplateData struct {
	Page  string
	Lang  string
	Files []DownloadFile
}

func Server(port string, sslPort string, certFile string, keyFile string, domain string, email string) error {
	// If domain is set and cert/key do not exist, obtain them using utils
	if domain != "" && email != "" && (certFile == "" || keyFile == "" || !utils.FileExists(certFile) || !utils.FileExists(keyFile)) {
		slog.Info("No SSL certificate found, attempting to obtain one with lego", "domain", domain)
		certFile = "cert.pem"
		keyFile = "key.pem"
		err := utils.ObtainCertWithLego(domain, email, certFile, keyFile)
		if err != nil {
			slog.Error("Failed to obtain SSL certificate with lego", "err", err)
			return err
		}
		slog.Info("Successfully obtained SSL certificate with lego", "certFile", certFile, "keyFile", keyFile)
	}

	loadTemplates()
	http.HandleFunc("/", makeLangHandler("home.html"))
	http.HandleFunc("/home", makeLangHandler("home.html"))
	http.HandleFunc("/about", makeLangHandler("about.html"))
	http.HandleFunc("/news", newsHandler)
	http.HandleFunc("/calendar", calendarHandler)
	http.HandleFunc("/taichi", makeLangHandler("taichi.html"))
	http.HandleFunc("/impressum", makeLangHandler("impressum.html"))
	http.HandleFunc("/download", downloadHandler)

	imagesSub, err := fs.Sub(imagesFS, "static/images")
	if err != nil {
		slog.Error("failed to create images sub FS", "err", err)
		return err
	}
	http.Handle("/api/images/", http.StripPrefix("/api/images/", http.FileServer(http.FS(imagesSub))))

	downloadsSub, err := fs.Sub(downloadsFS, "static/downloads")
	if err != nil {
		slog.Error("failed to create downloads sub FS", "err", err)
		return err
	}
	http.Handle("/api/downloads/", http.StripPrefix("/api/downloads/", http.FileServer(http.FS(downloadsSub))))

	httpAddr := "0.0.0.0:" + port
	httpsAddr := "0.0.0.0:" + sslPort

	if certFile != "" && keyFile != "" && utils.FileExists(certFile) && utils.FileExists(keyFile) {
		slog.Info("Starting HTTPS server at https://" + httpsAddr)
		go func() {
			if err := http.ListenAndServeTLS(httpsAddr, certFile, keyFile, nil); err != nil {
				slog.Error("HTTPS server failed", "err", err)
			}
		}()
	}

	slog.Info("Server started at http://" + httpAddr)
	return http.ListenAndServe(httpAddr, nil)
}

func makeLangHandler(page string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := getLang(r)
		tmpl, ok := templatesByLang[lang]
		if !ok {
			slog.Error("template not found for language", "lang", lang)
			http.Error(w, "Template not found", http.StatusInternalServerError)
			return
		}
		data := TemplateData{
			Page:          strings.TrimSuffix(page, ".html"),
			Lang:          lang,
			CalWebcalURLs: calendarURLs,
		}
		slog.Debug("renderTemplate", "lang", lang, "page", page)
		if err := tmpl.ExecuteTemplate(w, page, data); err != nil {
			slog.Error("render template", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
