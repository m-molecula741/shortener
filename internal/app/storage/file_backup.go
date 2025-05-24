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
	records  []URLRecord
}

func NewFileBackup(filePath string) *FileBackup {
	return &FileBackup{
		filePath: filePath,
		records:  make([]URLRecord, 0),
	}
}

func (fb *FileBackup) Clear() error {
	fb.records = make([]URLRecord, 0)
	return nil
}

// SaveURL сохраняет URL в память
func (fb *FileBackup) SaveURL(uuid, shortURL, originalURL string) error {
	record := URLRecord{
		UUID:        uuid,
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	fb.records = append(fb.records, record)
	return fb.saveToFile()
}

// saveToFile сохраняет все записи в файл
func (fb *FileBackup) saveToFile() error {
	file, err := os.Create(fb.filePath)
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Добавляем отступы для читаемости
	if err := encoder.Encode(fb.records); err != nil {
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

	if err := json.Unmarshal(data, &fb.records); err != nil {
		return nil, fmt.Errorf("cannot unmarshal records: %w", err)
	}

	urls := make(map[string]string)
	for _, record := range fb.records {
		urls[record.ShortURL] = record.OriginalURL
	}

	return urls, nil
}
