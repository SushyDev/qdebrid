package helpers

import (
	"bufio"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

func processFile(fileHeader *multipart.FileHeader) (io.ReadCloser, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %v", err)
	}

	return file, nil
}

func splitString(input string) []string {
	var result []string
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}
	return result
}

func validateAndParseForm(r *http.Request) error {
	contentType := parseContentType(r)

	switch contentType {
	case "multipart/form-data":
		return r.ParseMultipartForm(0)
	case "application/x-www-form-urlencoded":
		return r.ParseForm()
	default:
		return fmt.Errorf("unsupported Content-Type: %s", contentType)
	}
}

func parseContentType(r *http.Request) string {
	contentHeader := r.Header.Get("Content-Type")
	parts := strings.Split(contentHeader, ";")

	return parts[0]
}
