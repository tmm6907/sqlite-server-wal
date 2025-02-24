package util

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

func ImportDBFile(c echo.Context, file *multipart.FileHeader, username string) error {
	src, err := file.Open()
	if err != nil {
		c.Logger().Error("unable to open file: ", err)
		return err
	}
	defer src.Close()
	dest, err := os.Create(fmt.Sprintf("db/users/%s/%s.db", username, strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))))
	if err != nil {
		c.Logger().Error("unable to create new file: ", err)
		return err
	}
	defer dest.Close()
	if _, err = io.Copy(dest, src); err != nil {
		c.Logger().Error("unable to movie file data: ", err)
		return err
	}
	return nil
}

func ExportDBFile(c echo.Context, dbPath string, dbName string, fileType string) (*os.File, error) {
	switch fileType {
	case "db":
		c.Logger().Debug(dbName, fileType)
		tmpFile, err := os.Create(fmt.Sprintf("%s.%s", dbName, fileType))
		if err != nil {
			return nil, fmt.Errorf("%s, %s : %v", dbName, fileType, err)
		}
		c.Logger().Debug(tmpFile.Name())
		db, err := os.Open(dbPath)
		if err != nil {
			return nil, err
		}
		if _, err = io.Copy(tmpFile, db); err != nil {
			return nil, err
		}
		return tmpFile, nil
	case "csv":
		return exportCSV(dbPath, dbName)
	}
	return nil, fmt.Errorf("unsupported export type: %s", fileType)
}

func exportCSV(dbPath string, dbName string) (*os.File, error) {
	csvDir := "temp/"
	if err := os.MkdirAll(csvDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	cmd := exec.Command("sqlite3", dbPath, "SELECT name FROM sqlite_master WHERE type='table';")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	tables := splitLines(string(output))
	for _, table := range tables {
		if table == "" {
			continue
		}
		csvFilePath := filepath.Join(csvDir, fmt.Sprintf("%s.csv", table))
		exportCmd := exec.Command("sqlite3", dbPath)
		exportCmd.Stdin = strings.NewReader(fmt.Sprintf(".headers on\n.mode csv\n.output %s\nSELECT * FROM %s;\n", csvFilePath, table))
		exportCmd.Stdout = os.Stdout
		exportCmd.Stderr = os.Stderr
		if err := exportCmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to export table %s: %w", table, err)
		}

	}

	zipFilePath := filepath.Join(csvDir, fmt.Sprintf("%s.zip", dbName))
	files, err := filepath.Glob(filepath.Join(csvDir, "*.csv"))
	if err != nil || len(files) == 0 {
		return nil, fmt.Errorf("no CSV files found to zip")
	}
	zipCmd := exec.Command("zip", append([]string{"-j", zipFilePath}, files...)...)
	if err := zipCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create zip: %w", err)
	}

	// Open the zip file to return
	return os.Open(zipFilePath)
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}
