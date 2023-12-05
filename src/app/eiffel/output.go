package eiffel

import (
	"encoding/csv"
	"fmt"
	"github.com/org-harmony/harmony/src/app/template/parser"
	"github.com/org-harmony/harmony/src/app/user"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// OutputWriter defines the common interface for all EIFFEL's output writers.
// Most prominently, this is the CSVWriter.
type OutputWriter interface {
	// WriteHeaderRow writes the header row to the output file. The header row contains the names of the columns.
	// This is most important for OutputWriter implementations that write to files.
	// The header row should be written only once per file.
	WriteHeaderRow() error
	// WriteRow writes a row to the output file. The row contains the values of the columns.
	// The values are taken from the parser.ParsingResult and the user.User.
	WriteRow(pr parser.ParsingResult, usr *user.User) error
}

// OutputCfg contains the configuration for the output of EIFFEL.
type OutputCfg struct {
	// BaseDir is the base directory for the output of EIFFEL.
	// Each requirement will be written to an output file lying in a (sub-)directory of the base directory.
	// EIFFEL will create the (sub-)directories if they do not exist.
	// BaseDir should be "files/eiffel".
	BaseDir string `toml:"base_dir" env:"EIFFEL_OUTPUT_BASE_DIR" hvalidate:"required"`
}

// CSVWriter is an OutputWriter that writes to a CSV-file.
// The CSV-file contains the following columns: Requirement, Date, Time, Template, Variant, Template Version, Author.
// The CSVWriter requires a file handle to the output file. Ensure sufficient permissions for the file.
type CSVWriter struct {
	file *os.File
}

// DirSearch searches the baseDir for (sub-)directories containing the query string.
// The search is case-insensitive. Returns a slice of matching (sub-)directories.
// If no (sub-)directories match the query, an empty slice is returned.
// The returned slice contains the relative path to the (sub-)directories from the baseDir.
func DirSearch(baseDir string, query string) ([]string, error) {
	var dirs []string
	queryLower := strings.ToLower(query)

	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		if path == baseDir {
			return nil
		}

		visiblePath := strings.TrimPrefix(path, baseDir+"/")
		if strings.Contains(strings.ToLower(d.Name()), queryLower) || strings.Contains(strings.ToLower(visiblePath), queryLower) {
			dirs = append(dirs, visiblePath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return dirs, nil
}

// FileSearch searches a specified subdirectory for .csv-files containing the query string in their name.
// Only files with the .csv-extension are considered. The search is case-insensitive. Returns a slice of matching files.
func FileSearch(dirPath string, query string) ([]string, error) {
	var files []string
	queryLower := strings.ToLower(query)

	err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		filenameExt := d.Name()
		ext := filepath.Ext(filenameExt)
		if strings.ToLower(ext) != ".csv" {
			return nil
		}

		name := strings.TrimSuffix(filenameExt, ext)
		if strings.Contains(strings.ToLower(name), queryLower) {
			files = append(files, name)
		}

		return nil
	})

	if err != nil && !os.IsNotExist(err) { // ignore "file/dir does not exist" errors - the search might be submitted before the file/dir is created
		return nil, err
	}

	return files, nil
}

// BuildDirPath takes in a base dir + sub-path and returns the sanitized, full dir-path.
// The sub-path can be empty. The sub-path is sanitized before being used in the dir-path.
// The baseDir is not sanitized! It is assumed to be safe.
// Every character that is not a letter, number, underscore or hyphen is replaced by an underscore.
func BuildDirPath(baseDir, subPath string) string {
	return filepath.Join(baseDir, filepath.Clean(SanitizeFilepath(subPath)))
}

// BuildFilename takes in a filename (w/o extension) and returns the sanitized filename with the .csv-extension.
func BuildFilename(filename string) string {
	return fmt.Sprintf("%s.csv", SanitizeFilename(filename))
}

// SanitizeFilename takes in a filename and returns the sanitized filename.
// Every character that is not a letter, number, underscore or hyphen is replaced by an underscore.
// Different from SanitizeFilepath, this function replaces slashes with underscores.
func SanitizeFilename(filename string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

	return reg.ReplaceAllString(filename, "_")
}

// SanitizeFilepath takes in a filepath and returns the sanitized filepath.
// Every character that is not a letter (a-z & A-Z), number (0-9), underscore, slash or hyphen is replaced by an underscore.
func SanitizeFilepath(path string) string {
	reg := regexp.MustCompile(`[^/a-zA-Z0-9_-]+`)

	return reg.ReplaceAllString(path, "_")
}

// CreateIfNotExists creates the specified file in a directory and all the necessary directories if they do not exist.
// The file is created with the specified permissions. If the file already exists, nothing happens.
// CreateIfNotExists returns the file, a boolean indicating whether the file was created and an error.
func CreateIfNotExists(dirPath, filename string, perm os.FileMode) (*os.File, bool, error) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, perm); err != nil {
			return nil, false, err
		}
	}

	filePath := filepath.Join(dirPath, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if file, err := os.Create(filePath); err == nil {
			return file, true, nil
		}
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_APPEND, perm)

	return file, false, err
}

// WriteHeaderRow writes the header row to the output file. The header row contains the names of the columns.
// The columns are: Requirement, Date, Time, Template, Variant, Template Version, Author.
func (c *CSVWriter) WriteHeaderRow() error {
	header := []string{"Requirement", "Date", "Time", "Template", "Variant", "Template Version", "Author"}
	writer := csv.NewWriter(c.file)

	err := writer.Write(header)
	if err != nil {
		return err
	}

	writer.Flush()

	return nil
}

// WriteRow writes a row to the output file. The row contains the values of the columns.
// The values are the requirement, the current date and time, the template name, the variant name, the template version and the author.
func (c *CSVWriter) WriteRow(pr parser.ParsingResult, usr *user.User) error {
	now := time.Now()
	row := []string{
		pr.Requirement,
		now.Format("2006-01-02"),
		now.Format("15:04:05"),
		pr.TemplateName,
		pr.VariantName,
		pr.TemplateVersion,
		fmt.Sprintf("%s %s", usr.Firstname, usr.Lastname),
	}

	writer := csv.NewWriter(c.file)

	err := writer.Write(row)
	if err != nil {
		return err
	}

	writer.Flush()

	return nil
}
