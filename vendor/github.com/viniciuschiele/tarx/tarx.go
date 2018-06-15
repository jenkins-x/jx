// Package tarx implements access to tar archives on top of archive/​tar.
package tarx

import (
	"archive/tar"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

// Compression is the state represents if compressed or not.
type Compression int

const (
	// Uncompressed represents the uncompressed.
	Uncompressed Compression = iota
	// Gzip is gzip compression algorithm.
	Gzip
	// Bzip2 is bzip2 compression algorithm.
	Bzip2
)

// Common errors
var (
	ErrAppendNotSupported = errors.New("Append is only supported on compressed files")
	ErrBzip2NotSupported  = errors.New("Bzip2 is not supported for compression")
)

// CompressOptions is the compression configuration
type CompressOptions struct {
	Append           bool
	Compression      Compression
	IncludeSourceDir bool
	Filters          []string
}

// ExtractOptions is the decompression configuration
type ExtractOptions struct {
	FlatDir    bool
	Filters    []string
	NoOverride bool
}

// Internal struct to hold all resources to read a tar file
type tarReader struct {
	io.ReadCloser
	file           *os.File
	fileName       string
	reader         *tar.Reader
	compressReader io.ReadCloser
	header         *tar.Header
}

// Internal struct to hold all resources to write a tar file
type tarWriter struct {
	io.WriteCloser
	file           *os.File
	fileName       string
	writer         *tar.Writer
	compressWriter io.WriteCloser
}

// Compress compress a source path into a tar file.
// All files will be relative to the tar file.
func Compress(fileName, srcPath string, options *CompressOptions) error {
	if options == nil {
		options = &CompressOptions{}
	}

	srcInfo, err := os.Lstat(srcPath)
	if err != nil {
		return err
	}

	writer, err := newWriter(fileName, options)
	if err != nil {
		return err
	}

	// Removes the last slash to avoid different behaviors when `srcPath` is a folder
	srcPath = path.Clean(srcPath)

	// All files added are relative to the tar file
	// If IncludeSourceDir is true one level behind is added
	relPath := path.Dir(srcPath)
	if srcInfo.IsDir() && !options.IncludeSourceDir {
		relPath = srcPath
	}

	// To improve performance filters are prepared before.
	filters := prepareFilters(options.Filters)

	err = filepath.Walk(srcPath,
		func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Makes the file to be relative to the tar file
			// We don't support absolute path while compressing
			// but it can be done further
			relFilePath, err := filepath.Rel(relPath, filePath)
			if err != nil {
				return err
			}

			// When IncludeSourceDir is false the relative path for the
			// root folder is '.', we have to ignore this folder
			if relFilePath == "." {
				return nil
			}

			// Check if we have to add the current file based on the user filters
			if !optimizedMatches(relFilePath, filters) {
				return nil
			}

			// All good, relative path made, filters applied, now we can write
			// the user file into tar file
			return writer.Write(filePath, relFilePath)
		})

	// If any error occurs we delete the tar file
	writer.Close(err != nil)

	return err
}

// Extract extracts the files from a tar file into a target directory.
func Extract(fileName, targetDir string, options *ExtractOptions) error {
	if options == nil {
		options = &ExtractOptions{}
	}

	reader, err := newReader(fileName)
	if err != nil {
		return err
	}

	defer reader.Close()

	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		return err
	}

	// To improve performance the filters are prepared before.
	filters := prepareFilters(options.Filters)

	for {
		err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Removes the last slash to avoid different behaviors when `header.Name` is a folder
		targetFileName := filepath.Clean(reader.header.Name)

		// Check if we have to extact the current file based on the user filters
		if !optimizedMatches(targetFileName, filters) {
			continue
		}

		// If FlatDir is true we have to extract all files into root folder
		// and we have to ignore all sub directories
		if options.FlatDir {
			if reader.header.Typeflag == tar.TypeDir {
				continue
			}
			targetFileName = filepath.Base(targetFileName)
		}

		// If `targetFileName` is an absolute path we are going to extract it
		// relative to the `targetDir`
		targetFileName = path.Join(targetDir, targetFileName)

		if err := reader.Extract(targetFileName, options.NoOverride); err != nil {
			return err
		}
	}
}

// Find returns the header and ReadCloser for the entry in the tarfile
// that matches the filename. If nothing matches, an `os.ErrNotExists`
// error is returned.
// If the `targetFileName` is not a regular file it returns a reader `nil`.
func Find(fileName, targetFileName string) (*tar.Header, io.ReadCloser, error) {
	reader, err := newReader(fileName)
	if err != nil {
		return nil, nil, err
	}

	targetFileName = path.Clean(targetFileName)

	for {
		header, err := reader.reader.Next()
		if err == io.EOF {
			reader.Close()
			return nil, nil, os.ErrNotExist
		}
		if err != nil {
			reader.Close()
			return nil, nil, err
		}

		// If the file found is not a regular file we don't return a reader
		if targetFileName == path.Clean(header.Name) {
			if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
				return header, reader, nil
			}
			reader.Close()
			return header, nil, nil
		}
	}
}

// List lists all entries from a tar file.
func List(fileName string) ([]*tar.Header, error) {
	reader, err := newReader(fileName)
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	headers := []*tar.Header{}

	for {
		err := reader.Next()
		if err == io.EOF {
			return headers, nil
		}
		if err != nil {
			return nil, err
		}

		headers = append(headers, reader.header)
	}
}

// newReader opens a tar file as readonly
func newReader(fileName string) (*tarReader, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	// Reads the header from the file to see which compression
	// this file has been using.
	compression, err := detectCompression(file)
	if err != nil {
		return nil, err
	}

	var compressReader io.ReadCloser

	switch compression {
	case Gzip:
		if compressReader, err = gzip.NewReader(file); err != nil {
			file.Close()
			return nil, err
		}
	case Bzip2:
		compressReader = &readCloserWrapper{Reader: bzip2.NewReader(file)}
	}

	var reader *tar.Reader

	if compressReader == nil {
		reader = tar.NewReader(file)
	} else {
		reader = tar.NewReader(compressReader)
	}

	return &tarReader{
		file:           file,
		fileName:       fileName,
		reader:         reader,
		compressReader: compressReader,
	}, nil
}

// newReader creates a new tar file on disk if `append=false` otherwise
// it opens the tar file.
func newWriter(fileName string, options *CompressOptions) (*tarWriter, error) {
	var file *os.File
	var err error

	if options.Append {
		file, err = os.OpenFile(fileName, os.O_RDWR, os.ModePerm)
	} else {
		file, err = os.Create(fileName)
	}

	if err != nil {
		return nil, err
	}

	// In case of error we close and remove the tar file
	// if it was just created (append=false)
	defer func() {
		if err != nil {
			file.Close()

			if !options.Append {
				os.Remove(fileName)
			}
		}
	}()

	compression := options.Compression

	if options.Append {
		// Reads the header from the file to see which compression
		// this file has been using.
		compression, err := detectCompression(file)
		if err != nil {
			return nil, err
		}

		// I have only found this hack to append files into a tar file.
		// It works only for uncompressed tar files :(
		// http://stackoverflow.com/questions/18323995/golang-append-file-to-an-existing-tar-archive
		// We may improve it in the future.

		if compression != Uncompressed {
			return nil, ErrAppendNotSupported
		}

		if _, err = file.Seek(-2<<9, os.SEEK_END); err != nil {
			file.Close()
			return nil, err
		}
	}

	var compressWriter io.WriteCloser

	switch compression {
	case Gzip:
		compressWriter = gzip.NewWriter(file)
	case Bzip2:
		return nil, ErrBzip2NotSupported
	}

	var writer *tar.Writer

	if compressWriter == nil {
		writer = tar.NewWriter(file)
	} else {
		writer = tar.NewWriter(compressWriter)
	}

	return &tarWriter{
		file:           file,
		writer:         writer,
		compressWriter: compressWriter,
	}, nil
}

// detectCompression detects which comperssion the tar file has been using.
func detectCompression(file *os.File) (Compression, error) {
	source := make([]byte, 3)

	if _, err := file.ReadAt(source, 0); err != nil {
		return Uncompressed, err
	}

	for compression, m := range map[Compression][]byte{
		Bzip2: {0x42, 0x5A, 0x68},
		Gzip:  {0x1F, 0x8B, 0x08},
	} {
		if len(source) < len(m) {
			continue
		}
		if bytes.Compare(m, source[:len(m)]) == 0 {
			return compression, nil
		}
	}
	return Uncompressed, nil
}

// Extract extracts a tar file into disk
func (r *tarReader) Extract(fileName string, noOverride bool) error {
	fileInfo, err := os.Lstat(fileName)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// If the `fileName` already exists on disk and it is a file
	// we try to delete it in order to create a new one unless
	// `noOverride` is set to true
	if err == nil && !fileInfo.IsDir() {
		if noOverride {
			return nil
		}

		if err := os.Remove(fileName); err != nil {
			return err
		}
	}

	// header.Mode is in linux format, we have to convert it to os.FileMode
	// to be compatible to other platforms.
	headerInfo := r.header.FileInfo()

	switch r.header.Typeflag {
	case tar.TypeDir:
		if err := os.Mkdir(fileName, headerInfo.Mode()); err != nil && !os.IsExist(err) {
			return err
		}
	case tar.TypeReg, tar.TypeRegA:
		if err := createFile(fileName, headerInfo.Mode(), r.reader); err != nil {
			return err
		}
	case tar.TypeSymlink:
		if err := os.Symlink(r.header.Linkname, fileName); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unhandled tar header type %d", r.header.Typeflag)
	}

	return nil
}

// Next is just a wrapper aroung tar.Reader.Next
func (r *tarReader) Next() error {
	header, err := r.reader.Next()
	r.header = header
	return err
}

// Next is just a wrapper aroung tar.Reader.Read
func (r *tarReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

// Close closes the tar file.
func (r *tarReader) Close() error {
	if r.compressReader != nil {
		if err := r.compressReader.Close(); err != nil {
			return err
		}
	}

	if err := r.file.Close(); err != nil {
		return err
	}

	return nil
}

// Close closes the tar file, we usually use remove=true when the tar file
// is created but the compression fails, in this case
// we have to delete the tar file.
func (w *tarWriter) Close(remove bool) error {
	if w.writer != nil {
		if err := w.writer.Close(); err != nil {
			return err
		}
	}

	if w.compressWriter != nil {
		if err := w.compressWriter.Close(); err != nil {
			return err
		}
	}

	if err := w.file.Close(); err != nil {
		return err
	}

	if remove {
		return os.Remove(w.fileName)
	}

	return nil
}

// Write writes a file from disk into a tar file.
func (w *tarWriter) Write(fileName, name string) error {
	fileInfo, err := os.Lstat(fileName)
	if err != nil {
		return err
	}

	link := ""
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		if link, err = os.Readlink(fileName); err != nil {
			return err
		}
	}

	header, err := tar.FileInfoHeader(fileInfo, link)
	if err != nil {
		return err
	}

	header.Name = name

	if err := w.writer.WriteHeader(header); err != nil {
		return err
	}

	if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
		return nil
	}

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}

	defer file.Close()

	_, err = io.Copy(w.writer, file)
	return err
}
