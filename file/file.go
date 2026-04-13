package file

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/pkg/errors"
)

const ONE_GB = 1024 * 1024 * 1024
const ONE_MB = 1024 * 1024

// 统计z指定目录下目录和文件的总数
func CountDirAndFile(dir string) (int64, error) {
	var count int64
	if err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}
		count++
		return nil
	}); err != nil {
		return 0, err
	}
	return count, nil
}

func Size(file string) int64 {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return -1
	}
	return fileInfo.Size()
}

func DirSize(path string) int64 {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return -1
	}
	return size
}

func Exists(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}

func IsDir(path string) bool {
	if s, err := os.Stat(path); err == nil {
		return s.IsDir()
	}
	return false
}

// 拷贝文件或者目录到指定的目录下
func CopyToDir(item string, destDir string) error {
	if IsDir(item) {
		newDir := filepath.Join(destDir, filepath.Base(item))
		if err := os.MkdirAll(newDir, 0755); err != nil {
			return errors.WithStack(err)
		}
		return CopyDir(item, newDir)
	}
	return CopyFile(item, filepath.Join(destDir, filepath.Base(item)))

}

func MoveFile(src, dst string) error {
	if filepath.VolumeName(src) != filepath.VolumeName(dst) {
		if err := CopyFile(src, dst); err != nil {
			return err
		}
		return os.Remove(src)
	}
	if src == dst {
		return nil
	}
	os.Remove(dst) // 先删除目标文件
	return os.Rename(src, dst)
}

func CopyFile(src, dst string) error {
	// 打开源文件（只读）
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close() // 确保最后关闭源文件

	// 创建目标文件（如果存在会截断）
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	defer dstFile.Close() // 确保最后关闭目标文件

	// 复制内容（io.Copy 会自动使用缓冲区，高效处理大文件）
	_, err = io.Copy(dstFile, srcFile)

	if err != nil {
		return err
	}

	// 可选：强制刷新目标文件到磁盘（确保数据持久化）
	// 注意：os.File 的 Sync 方法可能返回错误，需处理
	// dstFile.Sync()
	// if err := dstFile.Sync(); err != nil {
	// 	return err
	// }

	return nil
}

// 将src目录下的所有内容复制到dst目录下
func CopyDir(src, dst string) error {
	if !Exists(src) {
		return nil
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}
		return CopyFile(path, dstPath)
	})
}

// 递归查找指定后缀的文件
func Glob(dir string, extensions []string) ([]string, error) {
	var list []string

	if !Exists(dir) {
		return list, nil
	}
	// 使用 os.WalkDir 递归遍历目录
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return errors.WithStack(err)
		}

		// 跳过目录，仅处理文件
		if d.IsDir() {
			return nil
		}

		if extensions != nil {
			ext := filepath.Ext(path)
			if !slices.Contains(extensions, ext) {
				return nil
			}
		}
		list = append(list, path)
		return nil
	})

	if err != nil {
		return nil, errors.WithStack(err)
	}
	return list, nil
}
