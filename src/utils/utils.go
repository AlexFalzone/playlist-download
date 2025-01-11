package utils

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DownloadFile scarica una risorsa da URL e restituisce i byte
func DownloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer func(body io.ReadCloser) {
		if cErr := body.Close(); cErr != nil {
			// log or handle
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s: %w", url, err)
	}
	return data, nil
}

// RunCmd esegue un comando esterno con i relativi argomenti e ne ritorna l'output o un errore
func RunCmd(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, stderr.String())
	}
	return out.Bytes(), nil
}

func EnsureDefaultOutputDir(outputDir string) (string, error) {
	if outputDir != "" {
		return outputDir, nil
	}

	rootDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("unable to get current directory: %w", err)
	}

	defaultDir := filepath.Join(rootDir, "output")

	if _, err := os.Stat(defaultDir); os.IsNotExist(err) {
		if err := os.MkdirAll(defaultDir, 0755); err != nil {
			return "", fmt.Errorf("error creating default output directory: %w", err)
		}
	}

	return defaultDir, nil
}

// Retry attempts a given function multiple times with a delay between each attempt.
// maxRetries: quante volte ci riproviamo
// delay: quanto attendiamo tra un tentativo e l'altro
// f: la funzione da eseguire
func Retry(maxRetries int, delay time.Duration, f func() error) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = f()
		if err == nil {
			// Successo, interrompiamo
			return nil
		}
		// Se non è riuscito, aspettiamo un po' e riproviamo
		time.Sleep(delay)
	}
	return fmt.Errorf("all retries failed after %d attempts. Last error: %w", maxRetries, err)
}

func DownloadFileWithRetry(url string, maxRetries int, delay time.Duration) ([]byte, error) {
	var data []byte

	err := Retry(maxRetries, delay, func() error {
		// La logica del singolo tentativo
		resp, err := http.Get(url)
		if err != nil {
			return fmt.Errorf("failed to download from %s: %w", url, err)
		}
		defer func(Body io.ReadCloser) {
			if cErr := Body.Close(); cErr != nil {
				fmt.Println("error closing body")
			}
		}(resp.Body)

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("non-200 status code: %d", resp.StatusCode)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response from %s: %w", url, err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return data, nil
}

func RunCmdWithRetry(command string, args []string, maxRetries int, delay time.Duration) ([]byte, error) {
	var out []byte
	err := Retry(maxRetries, delay, func() error {
		cmd := exec.Command(command, args...)
		output, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			return fmt.Errorf("cmd failed: %v\noutput: %s", cmdErr, string(output))
		}
		out = output
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

var invalidCharsRegex = regexp.MustCompile(`[\\/:*?"<>|]`)

func RemoveIllegalPathChars(name string) string {
	return invalidCharsRegex.ReplaceAllString(name, "_")
}

func CleanTitleForSearch(raw string) string {
	reParens := regexp.MustCompile(`\([^)]*\)`)
	cleaned := reParens.ReplaceAllString(raw, "")

	reSpecial := regexp.MustCompile(`[“”"–]+`)
	cleaned = reSpecial.ReplaceAllString(cleaned, " ")

	reFeat := regexp.MustCompile(`(?i)\bfeat\.?\b`)
	cleaned = reFeat.ReplaceAllString(cleaned, "")

	reSpaces := regexp.MustCompile(`\s+`)
	cleaned = reSpaces.ReplaceAllString(cleaned, " ")

	return strings.TrimSpace(cleaned)
}
