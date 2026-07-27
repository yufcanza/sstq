package corpus

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ExtractTar(tarPath, destdir string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("Ошибка открытия архива %s: %v", tarPath, err)
	}
	defer file.Close()

	if err := os.MkdirAll(destdir, 0755); err != nil {
		return fmt.Errorf("Ошибка создания временной директории для архива: %v", err)
	}

	tr := tar.NewReader(file)
	cleanDest := filepath.Clean(destdir)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Ошибка чтения архива: %v", err)

		}
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			continue
		}
		targetPath := filepath.Join(destdir, header.Name)
		cleanTarget := filepath.Clean(targetPath)

		if !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) && cleanTarget != cleanDest {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("Ошибка создание папки: %v", err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("Ошибка создание папки: %v", err)
			}
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("Ошибка создания файла %s: %v", targetPath, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("не удалось записать файл %s: %v", targetPath, err)
			}
			outFile.Close()
		}
	}
	return nil
}
