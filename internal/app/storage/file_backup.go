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
	records  map[string]URLRecord // ключ - shortURL для быстрого поиска существующих записей
}

func NewFileBackup(filePath string) *FileBackup {
	return &FileBackup{
		filePath: filePath,
		records:  make(map[string]URLRecord),
	}
}

func (fb *FileBackup) Clear() error {
	fb.records = make(map[string]URLRecord)
	return nil
}

// SaveURL сохраняет URL в память
func (fb *FileBackup) SaveURL(uuid, shortURL, originalURL string) error {
	// Проверяем, существует ли уже запись с таким shortURL
	if existingRecord, exists := fb.records[shortURL]; exists {
		// Если URL изменился, обновляем его, сохраняя старый UUID
		if existingRecord.OriginalURL != originalURL {
			existingRecord.OriginalURL = originalURL
			fb.records[shortURL] = existingRecord
		}
	} else {
		// Создаем новую запись только для новых shortURL
		record := URLRecord{
			UUID:        uuid,
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		}
		fb.records[shortURL] = record
	}

	return fb.saveToFile()
}

// saveToFile сохраняет все записи в файл
func (fb *FileBackup) saveToFile() error {
	file, err := os.Create(fb.filePath)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer file.Close()

	// Преобразуем map в slice для сохранения
	records := make([]URLRecord, 0, len(fb.records))
	for _, record := range fb.records {
		records = append(records, record)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Добавляем отступы для читаемости
	if err := encoder.Encode(records); err != nil {
		return fmt.Errorf("cannot encode records: %w", err)
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

	if len(data) == 0 {
		return make(map[string]string), nil
	}

	var records []URLRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("cannot unmarshal records: %w", err)
	}

	// Сохраняем записи в память
	for _, record := range records {
		fb.records[record.ShortURL] = record
	}

	// Преобразуем в map для возврата
	urls := make(map[string]string)
	for shortURL, record := range fb.records {
		urls[shortURL] = record.OriginalURL
	}

	return urls, nil
}
