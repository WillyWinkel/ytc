package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/WillyWinkel/ytc/internal/app"
	"github.com/WillyWinkel/ytc/internal/utils"
	"github.com/inconshreveable/go-update"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

var (
	port        string
	logfile     string
	Version     = "dev"
	showVersion bool
)

const (
	repoOwner     = "WillyWinkel"
	repoName      = "ytc"
	checkInterval = 6 * time.Hour
)

type program struct{}

func (p *program) Start(s service.Service) error {
	go func() {
		utils.SetupLogging(logfile)
		go periodicUpdateCheck()
		err := app.Server(port)
		if err != nil {
			slog.Error("failed to run server", "err", err.Error())
			os.Exit(1)
		}
	}()
	return nil
}
func (p *program) Stop(s service.Service) error { return nil }

var rootCmd = &cobra.Command{
	Use:   "ytc-server",
	Short: "YTC Server",
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Println(Version)
			os.Exit(0)
		}
		utils.SetupLogging(logfile)
		go periodicUpdateCheck()
		err := app.Server(port)
		if err != nil {
			slog.Error("failed to run server", "err", err.Error())
			os.Exit(1)
		}
	},
}

func Execute() {
	rootCmd.Flags().StringVarP(&port, "port", "p", "80", "Port to run the server on")
	rootCmd.Flags().StringVar(&logfile, "logfile", "", "Log file path pattern (enables file logging with rotation)")
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "Show version and exit")

	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(updateCmd())

	if err := rootCmd.Execute(); err != nil {
		slog.Error("command execution failed", "err", err.Error())
		os.Exit(1)
	}
}

func installCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install ytc-server as a system service",
		Run: func(cmd *cobra.Command, args []string) {
			exePath, err := os.Executable()
			if err != nil {
				fmt.Println("Could not determine executable path:", err)
				os.Exit(1)
			}
			svcConfig := &service.Config{
				Name:        "ytc-server",
				DisplayName: "YTC Server",
				Description: "YTC Server Service",
				Arguments:   []string{"--port", port, "--logfile", logfile},
				Executable:  exePath,
			}
			prg := &program{}
			s, err := service.New(prg, svcConfig)
			if err != nil {
				fmt.Println("Failed to create service:", err)
				os.Exit(1)
			}
			if err := s.Install(); err != nil {
				fmt.Println("Failed to install service:", err)
				os.Exit(1)
			}
			fmt.Println("Service installed successfully.")
			if err := s.Start(); err != nil {
				fmt.Println("Failed to start service:", err)
				os.Exit(1)
			}
			fmt.Println("Service started.")
		},
	}
}

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update ytc-server to the latest release from GitHub",
		Run: func(cmd *cobra.Command, args []string) {
			if err := updateSelf(); err != nil {
				fmt.Println("Update failed:", err)
				os.Exit(1)
			}
			fmt.Println("Update successful.")
		},
	}
}

func periodicUpdateCheck() {
	for {
		time.Sleep(checkInterval)
		if err := updateSelf(); err != nil {
			slog.Error("Periodic update failed", "err", err)
		}
	}
}

func updateSelf() error {
	latest, url, err := getLatestRelease()
	if err != nil {
		return err
	}
	current := getCurrentVersion()
	if latest == "" || latest == current {
		return nil
	}
	slog.Info("Updating to new version", "version", latest)
	tmpFile := filepath.Join(os.TempDir(), "ytc-server-update")
	if err := downloadFile(url, tmpFile); err != nil {
		return err
	}
	oldPath, err := backupCurrentBinary()
	if err != nil {
		return err
	}
	exePath, _ := os.Executable()
	f, err := os.Open(tmpFile)
	if err != nil {
		return err
	}
	defer f.Close()
	err = update.Apply(f, update.Options{})
	if err != nil {
		// rollback
		_ = restoreBackupBinary(oldPath, exePath)
		return fmt.Errorf("update failed, rolled back: %w", err)
	}
	slog.Info("Update applied, restarting...")
	restartSelf()
	return nil
}

func getLatestRelease() (version, assetURL string, err error) {
	api := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	resp, err := http.Get(api)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	var data struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}
	binName := binaryName()
	for _, asset := range data.Assets {
		if asset.Name == binName {
			return data.TagName, asset.BrowserDownloadURL, nil
		}
	}
	return "", "", fmt.Errorf("no suitable binary found in release")
}

func binaryName() string {
	arch := runtime.GOARCH
	osys := runtime.GOOS
	switch osys {
	case "linux":
		return "ytc-server-linux-" + arch
	case "windows":
		return "ytc-server-windows-" + arch + ".exe"
	case "darwin":
		return "ytc-server-darwin-" + arch
	}
	return "ytc-server"
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func backupCurrentBinary() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	backupPath := exePath + ".bak"
	input, err := os.Open(exePath)
	if err != nil {
		return "", err
	}
	defer input.Close()
	output, err := os.Create(backupPath)
	if err != nil {
		return "", err
	}
	defer output.Close()
	_, err = io.Copy(output, input)
	return backupPath, err
}

func restoreBackupBinary(backupPath, exePath string) error {
	input, err := os.Open(backupPath)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.Create(exePath)
	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, input)
	return err
}

func restartSelf() {
	exePath, _ := os.Executable()
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	os.Exit(0)
}

func getCurrentVersion() string {
	return Version
}
