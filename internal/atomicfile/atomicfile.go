package atomicfile

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func WriteFile(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("Ошибка создания директории %s: %w", dir, err)
	}
	tmpFile, err := os.CreateTemp(dir, "tmp-*")
	if err != nil {
		return fmt.Errorf("Ошибка создания временного файла: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		if err != nil {
			os.Remove(tmpName)
		}
	}()
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("Ошибка записи данных: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("Ошибка сихронизации: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("Ошибка закрытия временного файла: %w", err)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("Ошибка установки прав: %w", err)
	}
	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("Ошибка переименования: %w", err)
	}
	return nil
}

func WriteReader(filename string, reader io.Reader, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("Ошибка создания директории %s: %w", dir, err)
	}
	tmpFile, err := os.CreateTemp(dir, "tmp-*")
	if err != nil {
		return fmt.Errorf("Ошибка создания временного файла: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		if err != nil {
			os.Remove(tmpName)
		}
	}()
	if _, err := io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return fmt.Errorf("Ошибка копирования данных: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("Ошибка сихронизации: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("Ошибка закрытия временного файла: %w", err)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("Ошибка установки прав: %w", err)
	}
	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("Ошибка переименования: %w", err)
	}
	return nil

}
func WriteString(filename, content string, perm os.FileMode) error {
	return WriteFile(filename, []byte(content), perm)
}
