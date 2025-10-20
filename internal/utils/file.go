package utils

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/company/smartticket/internal/errors"
)

// File utilities

// FileExists checks if a file or directory exists
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// CreateDir creates a directory with all necessary parents
func CreateDir(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return errors.NewInternalError("Failed to create directory", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	return nil
}

// EnsureDir ensures a directory exists, creates it if necessary
func EnsureDir(path string) error {
	if !DirExists(path) {
		return CreateDir(path, 0755)
	}
	return nil
}

// ReadFile reads entire file into memory
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.NewNotFoundError("File not found").WithDetails(fmt.Sprintf("Path: %s", path))
		}
		return nil, errors.NewInternalError("Failed to read file", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	return data, nil
}

// WriteFile writes data to file, creating directory if necessary
func WriteFile(path string, data []byte, perm os.FileMode) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return errors.NewInternalError("Failed to write file", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	return nil
}

// AppendFile appends data to file, creating it if necessary
func AppendFile(path string, data []byte, perm os.FileMode) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return errors.NewInternalError("Failed to open file for append", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	defer func() { _ = file.Close() }()

	if _, err := file.Write(data); err != nil {
		return errors.NewInternalError("Failed to append to file", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	return nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return errors.NewInternalError("Failed to open source file", err).WithDetails(fmt.Sprintf("Source: %s", src))
	}
	defer func() { _ = sourceFile.Close() }()

	// Ensure destination directory exists
	dir := filepath.Dir(dst)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return errors.NewInternalError("Failed to create destination file", err).WithDetails(fmt.Sprintf("Destination: %s", dst))
	}
	defer func() { _ = destFile.Close() }()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return errors.NewInternalError("Failed to copy file", err).WithDetails(fmt.Sprintf("Source: %s, Destination: %s", src, dst))
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return errors.NewInternalError("Failed to get source file info", err)
	}

	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return errors.NewInternalError("Failed to set file permissions", err).WithDetails(fmt.Sprintf("Destination: %s", dst))
	}

	return nil
}

// MoveFile moves a file from src to dst
func MoveFile(src, dst string) error {
	if err := CopyFile(src, dst); err != nil {
		return err
	}

	if err := os.Remove(src); err != nil {
		return errors.NewInternalError("Failed to remove source file", err).WithDetails(fmt.Sprintf("Source: %s", src))
	}

	return nil
}

// DeleteFile deletes a file
func DeleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return errors.NewNotFoundError("File not found").WithDetails(fmt.Sprintf("Path: %s", path))
		}
		return errors.NewInternalError("Failed to delete file", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	return nil
}

// DeleteDir deletes a directory and all its contents
func DeleteDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return errors.NewInternalError("Failed to delete directory", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	return nil
}

// FileSize returns the size of a file in bytes
func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, errors.NewNotFoundError("File not found").WithDetails(fmt.Sprintf("Path: %s", path))
		}
		return 0, errors.NewInternalError("Failed to get file info", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	return info.Size(), nil
}

// FileHash calculates SHA256 hash of a file
func FileHash(path string) (string, error) {
	data, err := ReadFile(path)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// ListFiles returns a list of files in a directory
func ListFiles(dirPath string, recursive bool) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			relPath, err := filepath.Rel(dirPath, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		} else if !recursive && path != dirPath {
			return fs.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil, errors.NewInternalError("Failed to list files", err).WithDetails(fmt.Sprintf("Directory: %s", dirPath))
	}

	return files, nil
}

// ListDirs returns a list of directories in a directory
func ListDirs(dirPath string, recursive bool) ([]string, error) {
	var dirs []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && path != dirPath {
			relPath, err := filepath.Rel(dirPath, path)
			if err != nil {
				return err
			}
			dirs = append(dirs, relPath)

			if !recursive {
				return fs.SkipDir
			}
		}

		return nil
	})

	if err != nil {
		return nil, errors.NewInternalError("Failed to list directories", err).WithDetails(fmt.Sprintf("Directory: %s", dirPath))
	}

	return dirs, nil
}

// FileModTime returns the modification time of a file
func FileModTime(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, errors.NewNotFoundError("File not found").WithDetails(fmt.Sprintf("Path: %s", path))
		}
		return 0, errors.NewInternalError("Failed to get file info", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	return info.ModTime().Unix(), nil
}

// IsDir checks if a path is a directory
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile checks if a path is a file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GetFileExtension returns the file extension
func GetFileExtension(path string) string {
	return strings.ToLower(filepath.Ext(path))
}

// GetFileName returns the file name without extension
func GetFileName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// GetMimeType returns the MIME type of a file
func GetMimeType(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", errors.NewInternalError("Failed to open file", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	defer func() { _ = file.Close() }()

	// Read first 512 bytes to determine MIME type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", errors.NewInternalError("Failed to read file", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}

	mimeType := http.DetectContentType(buffer)
	return mimeType, nil
}

// IsImage checks if a file is an image based on MIME type
func IsImage(path string) bool {
	mimeType, err := GetMimeType(path)
	if err != nil {
		return false
	}
	return strings.HasPrefix(mimeType, "image/")
}

// IsText checks if a file is a text file based on MIME type
func IsText(path string) bool {
	mimeType, err := GetMimeType(path)
	if err != nil {
		return false
	}
	return strings.HasPrefix(mimeType, "text/")
}

// ReadLines reads a file line by line
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.NewInternalError("Failed to open file", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	defer func() { _ = file.Close() }()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, errors.NewInternalError("Failed to read file lines", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}

	return lines, nil
}

// WriteLines writes lines to a file
func WriteLines(path string, lines []string) error {
	file, err := os.Create(path)
	if err != nil {
		return errors.NewInternalError("Failed to create file", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	defer func() { _ = file.Close() }()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return errors.NewInternalError("Failed to write line", err).WithDetails(fmt.Sprintf("Path: %s", path))
		}
	}

	if err := writer.Flush(); err != nil {
		return errors.NewInternalError("Failed to flush writer", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}

	return nil
}

// TempFile creates a temporary file
func TempFile(dir, pattern string) (*os.File, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, errors.NewInternalError("Failed to create temp file", err)
	}
	return file, nil
}

// TempDir creates a temporary directory
func TempDir(dir, pattern string) (string, error) {
	path, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", errors.NewInternalError("Failed to create temp directory", err)
	}
	return path, nil
}

// FileUpload handles multipart file uploads
type FileUpload struct {
	File     multipart.File
	Header   *multipart.FileHeader
	Filename string
	Size     int64
	MimeType string
}

// HandleFileUpload processes a multipart file upload
func HandleFileUpload(fileHeader *multipart.FileHeader, maxFileSize int64) (*FileUpload, error) {
	if fileHeader == nil {
		return nil, errors.NewValidationError("No file provided")
	}

	if fileHeader.Size > maxFileSize {
		return nil, errors.NewValidationError("File too large").WithDetails(fmt.Sprintf("Max size: %d bytes", maxFileSize))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, errors.NewInternalError("Failed to open uploaded file", err)
	}

	// Detect MIME type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		_ = file.Close()
		return nil, errors.NewInternalError("Failed to read uploaded file", err)
	}

	mimeType := http.DetectContentType(buffer)

	// Reset file pointer
	if _, err := file.Seek(0, 0); err != nil {
		_ = file.Close()
		return nil, errors.NewInternalError("Failed to reset file pointer", err)
	}

	return &FileUpload{
		File:     file,
		Header:   fileHeader,
		Filename: fileHeader.Filename,
		Size:     fileHeader.Size,
		MimeType: mimeType,
	}, nil
}

// SaveUpload saves an uploaded file to the specified path
func SaveUpload(upload *FileUpload, path string) error {
	defer func() { _ = upload.File.Close() }()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	// Create destination file
	destFile, err := os.Create(path)
	if err != nil {
		return errors.NewInternalError("Failed to create destination file", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	defer func() { _ = destFile.Close() }()

	// Copy uploaded file to destination
	if _, err := io.Copy(destFile, upload.File); err != nil {
		return errors.NewInternalError("Failed to save uploaded file", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}

	return nil
}

// ValidateFileType validates file type based on allowed extensions
func ValidateFileType(filename string, allowedExtensions []string) error {
	ext := GetFileExtension(filename)
	if ext == "" {
		return errors.NewValidationError("File has no extension")
	}

	for _, allowedExt := range allowedExtensions {
		if ext == strings.ToLower(allowedExt) {
			return nil
		}
	}

	return errors.NewValidationError("File type not allowed").WithDetails(fmt.Sprintf("Allowed: %v", allowedExtensions))
}

// ValidateFileSize validates file size
func ValidateFileSize(size int64, maxSize int64) error {
	if size > maxSize {
		return errors.NewValidationError("File too large").WithDetails(fmt.Sprintf("Max size: %d bytes", maxSize))
	}
	return nil
}

// SanitizeFilename sanitizes a filename by removing dangerous characters
func SanitizeFilename(filename string) string {
	// Replace dangerous characters
	dangerous := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := filename

	for _, char := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Remove leading and trailing dots and spaces
	sanitized = strings.Trim(sanitized, ". ")

	// Ensure filename is not empty
	if sanitized == "" {
		sanitized = "file"
	}

	return sanitized
}

// CleanPath cleans a file path
func CleanPath(path string) string {
	return filepath.Clean(path)
}

// JoinPath joins path elements
func JoinPath(elements ...string) string {
	return filepath.Join(elements...)
}

// SplitPath splits a path into directory and file components
func SplitPath(path string) (dir, file string) {
	return filepath.Split(path)
}

// AbsPath returns the absolute path
func AbsPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.NewInternalError("Failed to get absolute path", err).WithDetails(fmt.Sprintf("Path: %s", path))
	}
	return absPath, nil
}

// RelativePath returns the relative path from base to target
func RelativePath(base, target string) (string, error) {
	relPath, err := filepath.Rel(base, target)
	if err != nil {
		return "", errors.NewInternalError("Failed to get relative path", err).WithDetails(fmt.Sprintf("Base: %s, Target: %s", base, target))
	}
	return relPath, nil
}
