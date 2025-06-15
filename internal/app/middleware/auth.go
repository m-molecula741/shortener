package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	CookieName       = "user_id"
	CookieExpiration = 24 * time.Hour
)

var (
	ErrInvalidCookie = errors.New("invalid cookie")
)

// AuthMiddleware middleware для аутентификации пользователей
type AuthMiddleware struct {
	gcm cipher.AEAD
}

// NewAuthMiddleware создает новый middleware для аутентификации
func NewAuthMiddleware(secretKey string) (*AuthMiddleware, error) {
	// Создаем ключ из строки (должен быть 32 байта для AES-256)
	key := make([]byte, 32)
	copy(key, []byte(secretKey))

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &AuthMiddleware{gcm: gcm}, nil
}

// Middleware обрабатывает аутентификацию пользователей
func (a *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := a.GetUserID(r)
		if err != nil {
			// Если куки нет или она невалидна, создаем новую
			userID = uuid.New().String()
			if err := a.SetUserID(w, userID); err != nil {
				http.Error(w, "Failed to set user cookie", http.StatusInternalServerError)
				return
			}
		}

		// Добавляем userID в контекст запроса
		ctx := r.Context()
		ctx = SetUserIDToContext(ctx, userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetUserID извлекает ID пользователя из куки
func (a *AuthMiddleware) GetUserID(r *http.Request) (string, error) {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return "", err
	}

	// Расшифровываем куку
	userID, err := a.decrypt(cookie.Value)
	if err != nil {
		return "", err
	}

	return userID, nil
}

// SetUserID устанавливает ID пользователя в куку
func (a *AuthMiddleware) SetUserID(w http.ResponseWriter, userID string) error {
	encryptedValue, err := a.encrypt(userID)
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     "user_id",
		Value:    encryptedValue,
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
	return nil
}

// encrypt шифрует строку
func (a *AuthMiddleware) encrypt(plaintext string) (string, error) {
	nonce := make([]byte, a.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := a.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// decrypt расшифровывает строку
func (a *AuthMiddleware) decrypt(ciphertext string) (string, error) {
	data, err := hex.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	nonceSize := a.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext_bytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := a.gcm.Open(nil, nonce, ciphertext_bytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
