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
// Only files with a matching .info file are included.
// If no .info file is found, use the filename as the description.
func getDownloadFiles(downloadDir string) ([]DownloadFile, error) {
	slog.Info("Scanning downloads directory", "dir", downloadDir)
	files, err := os.ReadDir(downloadDir)
	if err != nil {
		slog.Error("Failed to read downloads directory", "dir", downloadDir, "err", err)
		return nil, err
	}

	// Map base name to info file
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
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) == ".info" {
			continue
		}
		base := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		infoFileName, hasInfo := infoFiles[base]
		desc := ""
		if hasInfo {
			descPath := filepath.Join(downloadDir, infoFileName)
			if b, err := os.ReadFile(descPath); err == nil {
				desc = string(b)
				slog.Debug("Loaded description", "file", file.Name(), "descFile", infoFileName)
			} else {
				slog.Warn("Could not read description file", "descFile", infoFileName, "err", err)
				desc = file.Name()
			}
		} else {
			slog.Debug("No .info file found, using filename as description", "file", file.Name())
			desc = file.Name()
		}
		downloads = append(downloads, DownloadFile{
			Name:        file.Name(),
			URL:         "/api/downloads/" + file.Name(),
			Description: desc,
		})
		slog.Info("Prepared download file", "file", file.Name(), "descFile", infoFileName)
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
		slog.Error("render template", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// downloadHandler is the HTTP handler for the download page.
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	slog.Info("Handling download page request", "lang", lang, "remoteAddr", r.RemoteAddr)
	tmpl, ok := templatesByLang[lang]
	if !ok {
		slog.Error("template not found for language", "lang", lang)
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
