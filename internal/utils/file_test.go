package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileExists(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test-file-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Test existing file
	assert.True(t, FileExists(tmpFile.Name()))

	// Test non-existing file
	assert.False(t, FileExists("/path/to/nonexistent/file.txt"))

	// Test directory (should be false)
	tmpDir, err := os.MkdirTemp("", "test-dir-")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()
	assert.False(t, FileExists(tmpDir))
}

func TestDirExists(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test-dir-")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test existing directory
	assert.True(t, DirExists(tmpDir))

	// Test non-existing directory
	assert.False(t, DirExists("/path/to/nonexistent/dir"))

	// Test file (should be false)
	tmpFile, err := os.CreateTemp("", "test-file-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	assert.False(t, DirExists(tmpFile.Name()))
}

func TestCreateDir(t *testing.T) {
	// Create directory
	testDir := filepath.Join(os.TempDir(), "test-create-dir")
	defer func() { _ = os.RemoveAll(testDir) }()

	err := CreateDir(testDir, 0755)
	assert.NoError(t, err)
	assert.True(t, DirExists(testDir))

	// Test nested directory creation
	nestedDir := filepath.Join(testDir, "nested", "deep")
	err = CreateDir(nestedDir, 0755)
	assert.NoError(t, err)
	assert.True(t, DirExists(nestedDir))
}

func TestEnsureDir(t *testing.T) {
	// Test creating new directory
	testDir := filepath.Join(os.TempDir(), "test-ensure-dir")
	defer func() { _ = os.RemoveAll(testDir) }()

	err := EnsureDir(testDir)
	assert.NoError(t, err)
	assert.True(t, DirExists(testDir))

	// Test existing directory (should not error)
	err = EnsureDir(testDir)
	assert.NoError(t, err)
	assert.True(t, DirExists(testDir))
}

func TestReadFile(t *testing.T) {
	// Create a test file
	content := []byte("Hello, World!")
	tmpFile, err := os.CreateTemp("", "test-read-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.Write(content)
	require.NoError(t, err)
	func() { _ = tmpFile.Close() }()

	// Test reading existing file
	readContent, err := ReadFile(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Test reading non-existing file
	_, err = ReadFile("/path/to/nonexistent/file.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "File not found")
}

func TestWriteFile(t *testing.T) {
	// Write to new file (directory should be created)
	content := []byte("Hello, World!")
	testFile := filepath.Join(os.TempDir(), "test-write", "test.txt")
	defer func() { _ = os.RemoveAll(filepath.Dir(testFile)) }()

	err := WriteFile(testFile, content, 0644)
	assert.NoError(t, err)
	assert.True(t, FileExists(testFile))

	// Verify content
	readContent, err := ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, content, readContent)
}

func TestAppendFile(t *testing.T) {
	// Create initial file
	initialContent := []byte("Initial content")
	testFile := filepath.Join(os.TempDir(), "test-append", "test.txt")
	defer func() { _ = os.RemoveAll(filepath.Dir(testFile)) }()

	err := WriteFile(testFile, initialContent, 0644)
	require.NoError(t, err)

	// Append content
	appendContent := []byte("\nAppended content")
	err = AppendFile(testFile, appendContent, 0644)
	assert.NoError(t, err)

	// Verify combined content
	readContent, err := ReadFile(testFile)
	assert.NoError(t, err)
	expectedContent := append(initialContent, appendContent...)
	assert.Equal(t, expectedContent, readContent)
}

func TestCopyFile(t *testing.T) {
	// Create source file
	sourceContent := []byte("Source file content")
	sourceFile, err := os.CreateTemp("", "source-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(sourceFile.Name()) }()

	_, err = sourceFile.Write(sourceContent)
	require.NoError(t, err)
	func() { _ = sourceFile.Close() }()

	// Copy file
	destFile := filepath.Join(os.TempDir(), "dest-*.txt")
	defer func() { _ = os.Remove(destFile) }()

	err = CopyFile(sourceFile.Name(), destFile)
	assert.NoError(t, err)
	assert.True(t, FileExists(destFile))

	// Verify content
	destContent, err := ReadFile(destFile)
	assert.NoError(t, err)
	assert.Equal(t, sourceContent, destContent)
}

func TestMoveFile(t *testing.T) {
	// Create source file
	sourceContent := []byte("Move me")
	sourceFile, err := os.CreateTemp("", "move-source-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(sourceFile.Name()) }()

	_, err = sourceFile.Write(sourceContent)
	require.NoError(t, err)
	func() { _ = sourceFile.Close() }()

	// Move file
	destFile := filepath.Join(os.TempDir(), "move-dest-*.txt")
	defer func() { _ = os.Remove(destFile) }()

	err = MoveFile(sourceFile.Name(), destFile)
	assert.NoError(t, err)
	assert.True(t, FileExists(destFile))
	assert.False(t, FileExists(sourceFile.Name()))

	// Verify content
	destContent, err := ReadFile(destFile)
	assert.NoError(t, err)
	assert.Equal(t, sourceContent, destContent)
}

func TestDeleteFile(t *testing.T) {
	// Create test file
	testFile, err := os.CreateTemp("", "delete-*.txt")
	require.NoError(t, err)
	func() { _ = testFile.Close() }()

	// Delete file
	err = DeleteFile(testFile.Name())
	assert.NoError(t, err)
	assert.False(t, FileExists(testFile.Name()))

	// Try to delete non-existing file
	err = DeleteFile("/path/to/nonexistent/file.txt")
	assert.Error(t, err)
}

func TestDeleteDir(t *testing.T) {
	// Create directory with content
	testDir := filepath.Join(os.TempDir(), "delete-test")
	defer func() { _ = os.RemoveAll(testDir) }()

	err := CreateDir(testDir, 0755)
	require.NoError(t, err)

	// Create a file inside the directory
	testFile := filepath.Join(testDir, "test.txt")
	err = WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Delete directory
	err = DeleteDir(testDir)
	assert.NoError(t, err)
	assert.False(t, DirExists(testDir))
}

func TestFileSize(t *testing.T) {
	// Create test file with known content
	content := []byte("This file has 34 bytes")
	testFile, err := os.CreateTemp("", "size-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(testFile.Name()) }()

	_, err = testFile.Write(content)
	require.NoError(t, err)
	func() { _ = testFile.Close() }()

	// Get file size
	size, err := FileSize(testFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)

	// Test non-existing file
	_, err = FileSize("/path/to/nonexistent/file.txt")
	assert.Error(t, err)
}

func TestFileHash(t *testing.T) {
	// Create test file
	content := []byte("Test content for hashing")
	testFile, err := os.CreateTemp("", "hash-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(testFile.Name()) }()

	_, err = testFile.Write(content)
	require.NoError(t, err)
	func() { _ = testFile.Close() }()

	// Get file hash
	hash, err := FileHash(testFile.Name())
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Equal(t, 64, len(hash)) // SHA256 hash length

	// Verify hash is consistent
	hash2, err := FileHash(testFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, hash, hash2)
}

func TestListFiles(t *testing.T) {
	// Create test directory structure
	testDir := filepath.Join(os.TempDir(), "list-test")
	defer func() { _ = os.RemoveAll(testDir) }()

	// Create files
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, filename := range files {
		err := WriteFile(filepath.Join(testDir, filename), []byte("content"), 0644)
		require.NoError(t, err)
	}

	// Create subdirectory
	subDir := filepath.Join(testDir, "subdir")
	err := CreateDir(subDir, 0755)
	require.NoError(t, err)
	err = WriteFile(filepath.Join(subDir, "subfile.txt"), []byte("subcontent"), 0644)
	require.NoError(t, err)

	// Test non-recursive listing
	nonRecursive, err := ListFiles(testDir, false)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(nonRecursive))

	// Test recursive listing
	recursive, err := ListFiles(testDir, true)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(recursive)) // 3 root files + 1 subdirectory file
}

func TestListDirs(t *testing.T) {
	// Create test directory structure
	testDir := filepath.Join(os.TempDir(), "list-dirs-test")
	defer func() { _ = os.RemoveAll(testDir) }()

	// Create subdirectories
	dirs := []string{"dir1", "dir2"}
	for _, dirname := range dirs {
		err := CreateDir(filepath.Join(testDir, dirname), 0755)
		require.NoError(t, err)
	}

	// Create nested directory
	nestedDir := filepath.Join(testDir, "dir1", "nested")
	err := CreateDir(nestedDir, 0755)
	require.NoError(t, err)

	// Create a file (should not be listed)
	err = WriteFile(filepath.Join(testDir, "file.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	// Test non-recursive listing
	nonRecursive, err := ListDirs(testDir, false)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(nonRecursive))

	// Test recursive listing
	recursive, err := ListDirs(testDir, true)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(recursive)) // 2 root dirs + 1 nested dir
}

func TestFileModTime(t *testing.T) {
	// Create test file
	testFile, err := os.CreateTemp("", "modtime-*.txt")
	require.NoError(t, err)
	func() { _ = testFile.Close() }()
	defer func() { _ = os.Remove(testFile.Name()) }()

	// Get modification time
	modTime, err := FileModTime(testFile.Name())
	assert.NoError(t, err)
	assert.Greater(t, modTime, int64(0))

	// Test non-existing file
	_, err = FileModTime("/path/to/nonexistent/file.txt")
	assert.Error(t, err)
}

func TestIsDir(t *testing.T) {
	// Create directory
	testDir, err := os.MkdirTemp("", "test-isdir")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(testDir) }()

	// Test directory
	assert.True(t, IsDir(testDir))

	// Create file
	testFile, err := os.CreateTemp("", "test-isdir-file")
	require.NoError(t, err)
	func() { _ = testFile.Close() }()
	defer func() { _ = os.Remove(testFile.Name()) }()

	// Test file
	assert.False(t, IsDir(testFile.Name()))

	// Test non-existing path
	assert.False(t, IsDir("/path/to/nonexistent"))
}

func TestIsFile(t *testing.T) {
	// Create file
	testFile, err := os.CreateTemp("", "test-isfile")
	require.NoError(t, err)
	func() { _ = testFile.Close() }()
	defer func() { _ = os.Remove(testFile.Name()) }()

	// Test file
	assert.True(t, IsFile(testFile.Name()))

	// Create directory
	testDir, err := os.MkdirTemp("", "test-isfile-dir")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(testDir) }()

	// Test directory
	assert.False(t, IsFile(testDir))

	// Test non-existing path
	assert.False(t, IsFile("/path/to/nonexistent"))
}

func TestGetFileExtension(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"/path/to/file.txt", ".txt"},
		{"/path/to/file.json", ".json"},
		{"/path/to/file", ""},
		{"/path/to/.hidden", ".hidden"},
		{"/path/to/file.tar.gz", ".gz"},
		{"file.txt", ".txt"},
		{"file", ""},
		{"FILE.TXT", ".txt"},
	}

	for _, tc := range testCases {
		result := GetFileExtension(tc.input)
		assert.Equal(t, tc.expected, result, "GetFileExtension should work correctly for: %s", tc.input)
	}
}

func TestGetFileName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"/path/to/file.txt", "file"},
		{"/path/to/file.json", "file"},
		{"/path/to/file", "file"},
		{"/path/to/.hidden", ""},
		{"/path/to/file.tar.gz", "file.tar"},
		{"file.txt", "file"},
		{"file", "file"},
		{"/path/to/", "to"},
	}

	for _, tc := range testCases {
		result := GetFileName(tc.input)
		assert.Equal(t, tc.expected, result, "GetFileName should work correctly for: %s", tc.input)
	}
}

func TestGetMimeType(t *testing.T) {
	// Create text file with UTF-8 BOM to ensure proper detection
	textFile, err := os.CreateTemp("", "test-text-*.txt")
	require.NoError(t, err)
	// Write UTF-8 BOM followed by text content to ensure text/plain detection
	textContent := []byte("\ufeffThis is a text file with UTF-8 content")
	func() { _, _ = textFile.Write(textContent) }()
	func() { _ = textFile.Close() }()
	defer func() { _ = os.Remove(textFile.Name()) }()

	mimeType, err := GetMimeType(textFile.Name())
	assert.NoError(t, err)
	// Accept both text/plain results (with or without charset)
	assert.True(t, strings.HasPrefix(mimeType, "text/plain"),
		"Expected text/plain MIME type, got: %s", mimeType)

	// Test non-existing file
	_, err = GetMimeType("/path/to/nonexistent/file.txt")
	assert.Error(t, err)
}

func TestIsImage(t *testing.T) {
	// Create text file (should not be image)
	textFile, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	func() { _ = textFile.Close() }()
	defer func() { _ = os.Remove(textFile.Name()) }()

	assert.False(t, IsImage(textFile.Name()))
}

func TestIsText(t *testing.T) {
	// Create text file with UTF-8 BOM to ensure proper detection
	textFile, err := os.CreateTemp("", "test-*.txt")
	require.NoError(t, err)
	// Write UTF-8 BOM followed by text content to ensure text/plain detection
	textContent := []byte("\ufeffThis is a text file with UTF-8 content")
	func() { _, _ = textFile.Write(textContent) }()
	func() { _ = textFile.Close() }()
	defer func() { _ = os.Remove(textFile.Name()) }()

	assert.True(t, IsText(textFile.Name()))
}

func TestReadLines(t *testing.T) {
	// Create test file with multiple lines
	lines := []string{"Line 1", "Line 2", "Line 3"}
	content := []byte(strings.Join(lines, "\n"))
	testFile, err := os.CreateTemp("", "test-lines-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(testFile.Name()) }()

	_, err = testFile.Write(content)
	require.NoError(t, err)
	func() { _ = testFile.Close() }()

	// Read lines
	readLines, err := ReadLines(testFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, lines, readLines)

	// Test empty file
	emptyFile, err := os.CreateTemp("", "test-empty-*.txt")
	require.NoError(t, err)
	func() { _ = emptyFile.Close() }()
	defer func() { _ = os.Remove(emptyFile.Name()) }()

	emptyLines, err := ReadLines(emptyFile.Name())
	assert.NoError(t, err)
	assert.Empty(t, emptyLines)

	// Test non-existing file
	_, err = ReadLines("/path/to/nonexistent/file.txt")
	assert.Error(t, err)
}

func TestWriteLines(t *testing.T) {
	lines := []string{"Line 1", "Line 2", "Line 3"}
	testFile := filepath.Join(os.TempDir(), "test-write-lines.txt")
	defer func() { _ = os.Remove(testFile) }()

	// Write lines
	err := WriteLines(testFile, lines)
	assert.NoError(t, err)
	assert.True(t, FileExists(testFile))

	// Verify content
	readLines, err := ReadLines(testFile)
	assert.NoError(t, err)
	assert.Equal(t, lines, readLines)
}

func TestTempFile(t *testing.T) {
	// Create temp file
	tempFile, err := TempFile("", "test-temp-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()
	defer func() { _ = tempFile.Close() }()

	assert.NotNil(t, tempFile)
	assert.True(t, strings.HasPrefix(filepath.Base(tempFile.Name()), "test-temp-"))
	assert.True(t, strings.HasSuffix(tempFile.Name(), ".txt"))

	// Write to temp file
	_, err = tempFile.WriteString("test content")
	assert.NoError(t, err)
}

func TestTempDir(t *testing.T) {
	// Create temp directory
	tempDir, err := TempDir("", "test-temp-dir-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	assert.NotNil(t, tempDir)
	assert.True(t, strings.HasPrefix(filepath.Base(tempDir), "test-temp-dir-"))
	assert.True(t, DirExists(tempDir))
}

func TestValidateFileType(t *testing.T) {
	allowedExts := []string{".txt", ".json", ".png"}

	// Test valid extensions
	assert.NoError(t, ValidateFileType("test.txt", allowedExts))
	assert.NoError(t, ValidateFileType("test.JSON", allowedExts))
	assert.NoError(t, ValidateFileType("image.png", allowedExts))

	// Test invalid extension
	assert.Error(t, ValidateFileType("test.pdf", allowedExts))

	// Test no extension
	assert.Error(t, ValidateFileType("testfile", allowedExts))
}

func TestValidateFileSize(t *testing.T) {
	// Test valid size
	assert.NoError(t, ValidateFileSize(1000, 2000))
	assert.NoError(t, ValidateFileSize(2000, 2000))

	// Test invalid size
	assert.Error(t, ValidateFileSize(3000, 2000))
}

func TestSanitizeFilename(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"normal_file.txt", "normal_file.txt"},
		{"file with spaces.txt", "file with spaces.txt"},
		{"file/with\\slashes.txt", "file_with_slashes.txt"},
		{"file:with*special?chars.txt", "file_with_special_chars.txt"},
		{"..", "_"},
		{"   ", "file"},
		{"", "file"},
		{"../dangerous/path.txt", "__dangerous_path.txt"},
	}

	for _, tc := range testCases {
		result := SanitizeFilename(tc.input)
		assert.Equal(t, tc.expected, result, "SanitizeFilename should work correctly for: %s", tc.input)
	}
}

func TestCleanPath(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"normal/path", "normal/path"},
		{"path/with/./trailing/", "path/with/trailing"},
		{"../parent", "../parent"},
		{"./current", "current"},
	}

	for _, tc := range testCases {
		result := CleanPath(tc.input)
		assert.Equal(t, tc.expected, result, "CleanPath should work correctly for: %s", tc.input)
	}
}

func TestJoinPath(t *testing.T) {
	testCases := []struct {
		elements []string
		expected string
	}{
		{[]string{"path", "to", "file.txt"}, filepath.Join("path", "to", "file.txt")},
		{[]string{"absolute/path", "/file.txt"}, filepath.Join("absolute/path", "/file.txt")},
		{[]string{"single"}, "single"},
	}

	for _, tc := range testCases {
		result := JoinPath(tc.elements...)
		assert.Equal(t, tc.expected, result, "JoinPath should work correctly for: %v", tc.elements)
	}
}

func TestSplitPath(t *testing.T) {
	dir, file := SplitPath("/path/to/file.txt")
	assert.Equal(t, "/path/to/", dir)
	assert.Equal(t, "file.txt", file)

	dir, file = SplitPath("/path/to/")
	assert.Equal(t, "/path/to/", dir)
	assert.Equal(t, "", file)

	dir, file = SplitPath("file.txt")
	assert.Equal(t, "", dir)
	assert.Equal(t, "file.txt", file)
}

func TestAbsPath(t *testing.T) {
	// Test relative path
	relativePath := "test.txt"
	absPath, err := AbsPath(relativePath)
	assert.NoError(t, err)
	assert.True(t, filepath.IsAbs(absPath))

	// Test non-existing path (should still work)
	_, err = AbsPath("/path/to/nonexistent")
	assert.NoError(t, err)
}

func TestRelativePath(t *testing.T) {
	// Create test directory structure
	testDir, err := os.MkdirTemp("", "relative-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(testDir) }()

	// Create subdirectory
	subDir := filepath.Join(testDir, "subdir")
	err = CreateDir(subDir, 0755)
	require.NoError(t, err)

	// Create file
	testFile := filepath.Join(subDir, "test.txt")
	err = WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Get relative path
	relPath, err := RelativePath(testDir, testFile)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("subdir", "test.txt"), relPath)
}

func TestHandleFileUpload(t *testing.T) {
	// Skip this test as it requires complex multipart setup
	// This would be tested in integration tests with actual HTTP uploads
	t.Skip("HandleFileUpload requires HTTP multipart setup - tested in integration")
}

func TestSaveUpload(t *testing.T) {
	// Skip this test as it requires complex multipart setup
	// This would be tested in integration tests with actual HTTP uploads
	t.Skip("SaveUpload requires HTTP multipart setup - tested in integration")
}
