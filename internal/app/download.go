package app

import (
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// getDownloadFiles scans the downloads directory and returns a slice of DownloadFile.
// If a .info file is present, its content is used as description; otherwise, the filename is used.
func getDownloadFiles(downloadDir string) ([]DownloadFile, error) {
	slog.Info("Scanning downloads directory", "dir", downloadDir)
	files, err := os.ReadDir(downloadDir)
	if err != nil {
		slog.Error("Failed to read downloads directory", "dir", downloadDir, "err", err)
		return nil, err
	}

	infoFiles := make(map[string]string)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) == ".info" {
			base := strings.TrimSuffix(file.Name(), ".info")
			infoFiles[base] = file.Name()
			slog.Debug("Found info file", "infoFile", file.Name())
		}
	}

	var downloads []DownloadFile
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) == ".info" {
			continue
		}
		base := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		desc := file.Name()
		if infoFileName, hasInfo := infoFiles[base]; hasInfo {
			descPath := filepath.Join(downloadDir, infoFileName)
			if b, err := os.ReadFile(descPath); err == nil {
				desc = strings.TrimSpace(string(b))
				slog.Debug("Loaded description", "file", file.Name(), "descFile", infoFileName)
			} else {
				slog.Warn("Could not read description file", "descFile", infoFileName, "err", err)
			}
		} else {
			slog.Debug("No .info file found, using filename as description", "file", file.Name())
		}
		downloads = append(downloads, DownloadFile{
			Name:        file.Name(),
			URL:         "/api/downloads/" + file.Name(),
			Description: desc,
		})
		slog.Info("Prepared download file", "file", file.Name(), "description", desc)
	}
	slog.Info("Completed scanning downloads", "count", len(downloads))
	return downloads, nil
}

// renderDownloadPage renders the download page with the given files.
func renderDownloadPage(w http.ResponseWriter, tmpl *template.Template, lang string, files []DownloadFile) {
	slog.Info("Rendering download page", "lang", lang, "fileCount", len(files))
	data := DownloadTemplateData{
		Page:  "download",
		Lang:  lang,
		Files: files,
	}
	if err := tmpl.ExecuteTemplate(w, "download.html", data); err != nil {
		slog.Error("Failed to render download template", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// downloadHandler is the HTTP handler for the download page.
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	slog.Info("Handling download page request", "lang", lang, "remoteAddr", r.RemoteAddr)
	tmpl, ok := templatesByLang[lang]
	if !ok {
		slog.Error("Template not found for language", "lang", lang)
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	files, err := getDownloadFiles("static/downloads")
	if err != nil {
		slog.Error("Failed to get download files", "err", err)
		http.Error(w, "Download directory not found", http.StatusInternalServerError)
		return
	}

	renderDownloadPage(w, tmpl, lang, files)
}
