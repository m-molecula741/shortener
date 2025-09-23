// Package middleware содержит middleware для HTTP-обработчиков
package middleware

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware проверяет, что IP-адрес клиента входит в доверенную подсеть
type TrustedSubnetMiddleware struct {
	trustedSubnet *net.IPNet
}

// NewTrustedSubnetMiddleware создает новый middleware для проверки доверенной подсети
func NewTrustedSubnetMiddleware(trustedSubnetCIDR string) (*TrustedSubnetMiddleware, error) {
	if trustedSubnetCIDR == "" {
		// Если доверенная подсеть не указана, запрещаем доступ для всех
		return &TrustedSubnetMiddleware{
			trustedSubnet: nil,
		}, nil
	}

	_, subnet, err := net.ParseCIDR(trustedSubnetCIDR)
	if err != nil {
		return nil, err
	}

	return &TrustedSubnetMiddleware{
		trustedSubnet: subnet,
	}, nil
}

// Handler возвращает HTTP middleware для проверки доверенной подсети
func (m *TrustedSubnetMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Если доверенная подсеть не настроена, запрещаем доступ
		if m.trustedSubnet == nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Получаем IP из заголовка X-Real-IP
		realIP := r.Header.Get("X-Real-IP")
		if realIP == "" {
			// Если заголовок X-Real-IP не найден, запрещаем доступ
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Парсим IP-адрес
		clientIP := net.ParseIP(realIP)
		if clientIP == nil {
			// Если IP-адрес некорректный, запрещаем доступ
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Проверяем, входит ли IP в доверенную подсеть
		if !m.trustedSubnet.Contains(clientIP) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// IP-адрес входит в доверенную подсеть, продолжаем обработку
		next.ServeHTTP(w, r)
	})
}
