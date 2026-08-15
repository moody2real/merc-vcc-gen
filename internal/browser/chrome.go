package browser

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

const versionsURL = "https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json"

func Resolve(configuredPath string) (string, error) {
	if configuredPath != "" && fileExists(configuredPath) {
		log.Info("using configured chrome", "path", configuredPath)
		return configuredPath, nil
	}

	if p := localInstall(); p != "" {
		log.Info("using installed chrome", "path", p)
		return p, nil
	}

	if p := systemChrome(); p != "" {
		log.Info("using system chrome", "path", p)
		return p, nil
	}

	log.Info("no chrome found, downloading chrome for testing...")
	return install()
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func KillStale(userDataDir string) {
	marker := filepath.Base(userDataDir)
	if marker == "" || marker == "." || marker == string(os.PathSeparator) {
		return
	}

	log.Info("clearing stale chrome instances", "profile", marker)
	switch runtime.GOOS {
	case "windows":
		script := fmt.Sprintf(`Get-CimInstance Win32_Process -Filter "Name='chrome.exe'" | Where-Object { $_.CommandLine -like '*%s*' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }`, marker)
		_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Run()
	default:
		_ = exec.Command("pkill", "-f", marker).Run()
	}

	for _, name := range []string{"SingletonLock", "SingletonSocket", "SingletonCookie"} {
		_ = os.Remove(filepath.Join(userDataDir, name))
	}
}

func installDir() string {
	exe, err := os.Executable()
	if err != nil {
		exe, _ = os.Getwd()
	}
	return filepath.Join(filepath.Dir(exe), "chrome")
}

func platform() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "386" {
			return "win32", nil
		}
		return "win64", nil
	case "linux":
		return "linux64", nil
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "mac-arm64", nil
		}
		return "mac-x64", nil
	default:
		return "", fmt.Errorf("unsupported os: %s", runtime.GOOS)
	}
}

func binaryRelPath(plat string) string {
	switch plat {
	case "win64", "win32":
		return filepath.Join("chrome-"+plat, "chrome.exe")
	case "linux64":
		return filepath.Join("chrome-"+plat, "chrome")
	default:
		return filepath.Join("chrome-"+plat, "Google Chrome for Testing.app", "Contents", "MacOS", "Google Chrome for Testing")
	}
}

func localInstall() string {
	plat, err := platform()
	if err != nil {
		return ""
	}
	p := filepath.Join(installDir(), binaryRelPath(plat))
	if fileExists(p) {
		return p
	}
	return ""
}

func systemChrome() string {
	var candidates []string
	switch runtime.GOOS {
	case "windows":
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LocalAppData"} {
			if base := os.Getenv(env); base != "" {
				candidates = append(candidates, filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"))
			}
		}
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	default:
		candidates = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	}

	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}

	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

type versionsResponse struct {
	Channels map[string]struct {
		Version   string `json:"version"`
		Downloads struct {
			Chrome []struct {
				Platform string `json:"platform"`
				URL      string `json:"url"`
			} `json:"chrome"`
		} `json:"downloads"`
	} `json:"channels"`
}

func install() (string, error) {
	plat, err := platform()
	if err != nil {
		return "", err
	}

	downloadURL, version, err := resolveDownloadURL(plat)
	if err != nil {
		return "", err
	}
	log.Info("downloading chrome", "version", version, "platform", plat)

	dir := installDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create install dir: %w", err)
	}

	zipPath := filepath.Join(dir, "chrome.zip")
	if err := downloadFile(downloadURL, zipPath); err != nil {
		return "", fmt.Errorf("download chrome: %w", err)
	}
	defer os.Remove(zipPath)

	log.Info("extracting chrome...")
	if err := unzip(zipPath, dir); err != nil {
		return "", fmt.Errorf("extract chrome: %w", err)
	}

	bin := filepath.Join(dir, binaryRelPath(plat))
	if runtime.GOOS != "windows" {
		if err := os.Chmod(bin, 0o755); err != nil {
			return "", fmt.Errorf("chmod chrome: %w", err)
		}
	}
	if !fileExists(bin) {
		return "", fmt.Errorf("chrome binary missing after extract: %s", bin)
	}

	log.Info("chrome installed", "path", bin)
	return bin, nil
}

func resolveDownloadURL(plat string) (string, string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(versionsURL)
	if err != nil {
		return "", "", fmt.Errorf("fetch versions: %w", err)
	}
	defer resp.Body.Close()

	var v versionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", "", fmt.Errorf("decode versions: %w", err)
	}

	channel, ok := v.Channels["Stable"]
	if !ok {
		return "", "", fmt.Errorf("stable channel not found")
	}
	for _, d := range channel.Downloads.Chrome {
		if d.Platform == plat {
			return d.URL, channel.Version, nil
		}
	}
	return "", "", fmt.Errorf("no download for platform %s", plat)
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}
		out.Close()
		rc.Close()
	}
	return nil
}
