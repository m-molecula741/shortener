package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"sync"
	"time"
)

// DeleteRequest представляет запрос на удаление URL
type DeleteRequest struct {
	UserID   string
	ShortIDs []string
}

type URLService struct {
	storage  URLStorage
	baseURL  string
	dbPinger DatabasePinger

	// Каналы для асинхронного удаления
	deleteChan chan DeleteRequest
	workerWG   sync.WaitGroup
}

func NewURLService(storage URLStorage, baseURL string, dbPinger DatabasePinger) *URLService {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}

	service := &URLService{
		storage:    storage,
		baseURL:    baseURL,
		dbPinger:   dbPinger,
		deleteChan: make(chan DeleteRequest, 100), // Буфер для 100 запросов
	}

	// Запускаем воркеры для обработки удаления
	service.startDeleteWorkers()

	return service
}

// startDeleteWorkers запускает горутины для обработки удаления URL
func (s *URLService) startDeleteWorkers() {
	const numWorkers = 3

	for i := 0; i < numWorkers; i++ {
		s.workerWG.Add(1)
		go s.deleteWorker()
	}
}

// deleteWorker обрабатывает запросы на удаление URL
func (s *URLService) deleteWorker() {
	defer s.workerWG.Done()

	// Создаем каналы для fanIn паттерна
	batchChan := make(chan []DeleteRequest, 10)

	// Горутина для сбора запросов в batch
	go s.batchCollector(batchChan)

	// Обрабатываем batch запросы
	for batch := range batchChan {
		s.processBatch(batch)
	}
}

// batchCollector собирает запросы на удаление в батчи для эффективной обработки
func (s *URLService) batchCollector(batchChan chan<- []DeleteRequest) {
	defer close(batchChan)

	const (
		maxBatchSize = 10
		batchTimeout = 100 * time.Millisecond
	)

	var batch []DeleteRequest
	timer := time.NewTimer(batchTimeout)
	timer.Stop()

	for {
		select {
		case req, ok := <-s.deleteChan:
			if !ok {
				// Канал закрыт, отправляем последний батч
				if len(batch) > 0 {
					batchChan <- batch
				}
				return
			}

			batch = append(batch, req)

			// Если первый элемент в батче, запускаем таймер
			if len(batch) == 1 {
				timer.Reset(batchTimeout)
			}

			// Если батч полный, отправляем его
			if len(batch) >= maxBatchSize {
				batchChan <- batch
				batch = nil
				timer.Stop()
			}

		case <-timer.C:
			// Таймаут - отправляем накопленный батч
			if len(batch) > 0 {
				batchChan <- batch
				batch = nil
			}
		}
	}
}

// processBatch обрабатывает батч запросов на удаление
func (s *URLService) processBatch(batch []DeleteRequest) {
	// Группируем запросы по пользователям для batch update
	userBatches := make(map[string][]string)

	for _, req := range batch {
		userBatches[req.UserID] = append(userBatches[req.UserID], req.ShortIDs...)
	}

	// Обновляем БД для каждого пользователя
	for userID, shortIDs := range userBatches {
		if err := s.storage.BatchDeleteUserURLs(context.Background(), userID, shortIDs); err != nil {
			_ = err
		}
	}
}

// DeleteUserURLs добавляет запрос на асинхронное удаление URL пользователя
func (s *URLService) DeleteUserURLs(userID string, shortIDs []string) error {
	if len(shortIDs) == 0 {
		return nil
	}

	// Отправляем запрос в канал для асинхронной обработки
	req := DeleteRequest{
		UserID:   userID,
		ShortIDs: shortIDs,
	}

	select {
	case s.deleteChan <- req:
		return nil
	default:
		// Канал заполнен, возвращаем ошибку
		return ErrDeleteChannelFull
	}
}

// Close закрывает сервис и ждет завершения всех воркеров
func (s *URLService) Close() {
	close(s.deleteChan)
	s.workerWG.Wait()
}

// Добавляем пул для строк
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(strings.Builder)
	},
}

func (s *URLService) Shorten(url string) (string, error) {
	shortID, err := generateShortID()
	if err != nil {
		return "", err
	}

	// Используем пул для построения URL
	builder := bufferPool.Get().(*strings.Builder)
	builder.Reset()
	defer bufferPool.Put(builder)

	builder.WriteString(s.baseURL)
	if !strings.HasSuffix(s.baseURL, "/") {
		builder.WriteString("/")
	}
	builder.WriteString(shortID)
	shortURL := builder.String()

	if err := s.storage.Save(shortID, url); err != nil {
		if conflictErr, isConflict := IsURLConflict(err); isConflict {
			return s.baseURL + conflictErr.ExistingShortURL, &ErrURLConflict{
				ExistingShortURL: s.baseURL + conflictErr.ExistingShortURL,
			}
		}
		return "", err
	}

	return shortURL, nil
}

// ShortenWithUser сокращает URL и связывает его с пользователем
func (s *URLService) ShortenWithUser(ctx context.Context, url, userID string) (string, error) {
	shortURL, err := s.Shorten(url)
	if err != nil {
		// Если это конфликт URL, возвращаем существующий URL
		if _, isConflict := IsURLConflict(err); isConflict {
			return shortURL, err // shortURL уже содержит полный URL с baseURL
		}
		return "", err
	}

	// Если URL успешно создан и у нас есть userID, связываем его с пользователем
	if userID != "" {
		// Извлекаем shortID из shortURL
		shortID := shortURL[len(s.baseURL):]

		urlPair := URLPair{
			ShortID:     shortID,
			OriginalURL: url,
			UserID:      userID,
		}

		// Обновляем запись с userID через SaveBatch
		if err := s.storage.SaveBatch(ctx, []URLPair{urlPair}); err != nil {
			_ = err
		}
	}

	return shortURL, nil
}

func (s *URLService) Expand(shortID string) (string, error) {
	return s.storage.Get(shortID)
}

// Оптимизируем генерацию ID
var idPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 6) // 6 байт дадут 8 символов в base64
	},
}

func generateShortID() (string, error) {
	b := idPool.Get().([]byte)
	defer idPool.Put(b)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b)[:8], nil
}

// PingDB проверяет соединение с базой данных
func (s *URLService) PingDB() error {
	if s.dbPinger == nil {
		return nil // если пингер не настроен, возвращаем nil
	}
	return s.dbPinger.Ping()
}

// ShortenBatch сокращает множество URL за одну операцию
func (s *URLService) ShortenBatch(ctx context.Context, requests []BatchShortenRequest) ([]BatchShortenResponse, error) {
	if len(requests) == 0 {
		return []BatchShortenResponse{}, nil
	}

	// Подготавливаем данные для batch сохранения
	urlPairs := make([]URLPair, len(requests))
	responses := make([]BatchShortenResponse, len(requests))

	for i, req := range requests {
		shortID, err := generateShortID()
		if err != nil {
			return nil, err
		}

		urlPairs[i] = URLPair{
			ShortID:     shortID,
			OriginalURL: req.OriginalURL,
		}

		responses[i] = BatchShortenResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      s.baseURL + shortID,
		}
	}

	// Сохраняем все URL одной операцией
	if err := s.storage.SaveBatch(ctx, urlPairs); err != nil {
		return nil, err
	}

	return responses, nil
}

// ShortenBatchWithUser сокращает множество URL за одну операцию с привязкой к пользователю
func (s *URLService) ShortenBatchWithUser(ctx context.Context, requests []BatchShortenRequest, userID string) ([]BatchShortenResponse, error) {
	if len(requests) == 0 {
		return []BatchShortenResponse{}, nil
	}

	// Подготавливаем данные для batch сохранения
	urlPairs := make([]URLPair, len(requests))
	responses := make([]BatchShortenResponse, len(requests))

	for i, req := range requests {
		shortID, err := generateShortID()
		if err != nil {
			return nil, err
		}

		urlPairs[i] = URLPair{
			ShortID:     shortID,
			OriginalURL: req.OriginalURL,
			UserID:      userID,
		}

		responses[i] = BatchShortenResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      s.baseURL + shortID,
		}
	}

	// Сохраняем все URL одной операцией
	if err := s.storage.SaveBatch(ctx, urlPairs); err != nil {
		return nil, err
	}

	return responses, nil
}

// GetUserURLs получает все URL пользователя
func (s *URLService) GetUserURLs(ctx context.Context, userID string) ([]UserURL, error) {
	return s.storage.GetUserURLs(ctx, userID)
}
