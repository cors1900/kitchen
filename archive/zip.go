package archive

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cors1900/kitchen/file"
	"github.com/pkg/errors"
)

type Progress func(index int, total int, filename string)

// 将zip包的内容解压到指定目录下
func Unzip(zipPath string, targetDir string, progress Progress) error {
	tempDir := filepath.Clean(targetDir) + ".unzipping"
	os.RemoveAll(tempDir)
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer archive.Close()
	defer os.RemoveAll(tempDir)

	count := len(archive.File)
	for index, f := range archive.File {
		if progress != nil {
			progress(index, count, f.Name)
		}
		filePath := filepath.Join(tempDir, f.Name)
		if !strings.HasPrefix(filePath, tempDir+string(os.PathSeparator)) {
			return errors.WithStack(err)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return errors.WithStack(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return errors.WithStack(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return errors.WithStack(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return errors.WithStack(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
	if file.Exists(targetDir) {
		return file.CopyDir(tempDir, targetDir)
	}
	return os.Rename(tempDir, targetDir)
}

// Zip 将指定的文件或目录压缩到 ZIP 文件
// 如果仅指定一个目录，则将该目录内的所有内容放到压缩包内
func Zip(sources []string, target string, progress Progress) (err error) {
	// 1. 创建 ZIP 文件
	var zipFile *os.File
	if zipFile, err = os.Create(target); err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		zipFile.Close()
		if err != nil {
			os.Remove(target)
		}
	}()

	var total int64
	var index int64

	// 2. 初始化 zip.Writer
	writer := zip.NewWriter(zipFile)
	defer writer.Close()

	addFileToZip := func(sourcePath string, archivePath string, info os.FileInfo, writer *zip.Writer) error {
		if info.IsDir() {
			return nil
		}

		if progress != nil {
			progress(int(index), int(total), archivePath)
		}

		// 4. 创建 ZIP 文件头（保留文件权限）
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return errors.WithStack(err)
		}
		header.Name = archivePath   // ZIP 中的文件名（相对路径）
		header.Method = zip.Deflate // 使用 DEFLATE 压缩算法（默认）
		header.SetMode(info.Mode()) // 保留文件权限

		// 5. 写入文件头到 ZIP
		writerHeader, err := writer.CreateHeader(header)
		if err != nil {
			return errors.WithStack(err)
		}

		// 6. 复制文件内容到 ZIP
		file, err := os.Open(sourcePath)
		if err != nil {
			return errors.WithStack(err)
		}
		defer file.Close()

		_, err = io.Copy(writerHeader, file)
		if err != nil {
			return errors.WithStack(err)
		}

		return nil
	}

	addDirToZip := func(sourceDir string, archiveDir string, writer *zip.Writer) error {
		return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.WithStack(err)
			}

			index++

			// 跳过目录（ZIP 会自动处理目录结构）
			if info.IsDir() {
				return nil
			}

			// 计算文件在 ZIP 中的相对路径（保留源目录结构）
			relPath, err := filepath.Rel(sourceDir, filePath)
			if err != nil {
				return errors.WithStack(err)
			}

			// 7. 调用 addFileToZip 函数添加文件到 ZIP
			if err := addFileToZip(
				filePath,
				filepath.Join(archiveDir, relPath),
				info, writer); err != nil {
				return errors.WithStack(err)
			}
			return nil
		})
	}

	if len(sources) == 1 && file.IsDir(sources[0]) {
		total, _ = file.CountDirAndFile(sources[0])

		if err = addDirToZip(sources[0], "", writer); err != nil {
			return errors.WithStack(err)
		}
		return nil
	}

	for _, source := range sources {
		if source == "" {
			continue
		}

		if file.IsDir(source) {
			n, _ := file.CountDirAndFile(source)
			total += n
			continue
		}
		total++
	}

	for _, source := range sources {
		if source == "" {
			continue
		}
		var err error
		if file.IsDir(source) {
			if err = addDirToZip(source,
				filepath.Base(source), writer); err != nil {
				return errors.WithStack(err)
			}
			continue
		}
		var info os.FileInfo
		info, err = os.Stat(source)
		if err != nil {
			return errors.WithStack(err)
		}
		index++

		if err = addFileToZip(source,
			filepath.Base(source), info, writer); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func IsWindowsOS() bool {
	switch runtime.GOOS {
	case "windows":
		return true
	default:
		return false
	}
}

func GetPathSeparator() string {
	switch runtime.GOOS {
	case "windows":
		return ";" // Windows 使用分号
	default:
		return ":" // 类 Unix 系统使用冒号（Linux、macOS 等）
	}
}

func EllipseText(str string, n int) string {
	size := len(str)
	if size > n {
		return "..." + str[size-n+3:]
	}
	return str
}
