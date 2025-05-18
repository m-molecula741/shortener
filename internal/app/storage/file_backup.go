package storage

import (
	"encoding/json"
	"fmt"
	"os"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type FileBackup struct {
	filePath string
}

func NewFileBackup(filePath string) *FileBackup {
	return &FileBackup{
		filePath: filePath,
	}
}

func (fb *FileBackup) Clear() error {
	return os.WriteFile(fb.filePath, []byte("[\n"), 0666)
}

// SaveURL сохраняет URL в файл
func (fb *FileBackup) SaveURL(uuid, shortURL, originalURL string) error {
	file, err := os.OpenFile(fb.filePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer file.Close()

	// Определяем, есть ли уже записи в файле
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot get file stats: %w", err)
	}

	record := URLRecord{
		UUID:        uuid,
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("cannot marshal record: %w", err)
	}

	// Если файл пустой, создаем его с начальной структурой
	if stat.Size() == 0 {
		if _, err := file.Write([]byte("[\n")); err != nil {
			return fmt.Errorf("cannot write initial bracket: %w", err)
		}
		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("cannot write to file: %w", err)
		}
	} else {
		// Перемещаемся в конец файла минус один символ (перед "]" или "\n")
		if _, err := file.Seek(-2, 2); err != nil {
			return fmt.Errorf("cannot seek file: %w", err)
		}

		// Проверяем, есть ли уже записи (ищем "]")
		buf := make([]byte, 1)
		if _, err := file.Read(buf); err != nil {
			return fmt.Errorf("cannot read file: %w", err)
		}

		// Если это первая запись (нашли "[")
		if buf[0] == '[' {
			if _, err := file.Write([]byte("\n")); err != nil {
				return fmt.Errorf("cannot write newline: %w", err)
			}
			if _, err := file.Write(data); err != nil {
				return fmt.Errorf("cannot write record: %w", err)
			}
		} else {
			// Добавляем запятую и новую запись
			if _, err := file.Write([]byte(",\n")); err != nil {
				return fmt.Errorf("cannot write separator: %w", err)
			}
			if _, err := file.Write(data); err != nil {
				return fmt.Errorf("cannot write record: %w", err)
			}
		}
	}

	// Добавляем закрывающую скобку
	if _, err := file.Write([]byte("\n]")); err != nil {
		return fmt.Errorf("cannot write closing bracket: %w", err)
	}

	return nil
}

func (fb *FileBackup) LoadURLs() (map[string]string, error) {
	data, err := os.ReadFile(fb.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("cannot read file: %w", err)
	}

	// Если файл пустой, возвращаем пустую карту
	if len(data) == 0 {
		return make(map[string]string), nil
	}

	var records []URLRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("cannot unmarshal records: %w", err)
	}

	urls := make(map[string]string)
	for _, record := range records {
		urls[record.ShortURL] = record.OriginalURL
	}

	return urls, nil
}
