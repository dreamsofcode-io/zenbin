package flash

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

const flashMessagesKey = "flash_message"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		flashMessages := make(map[string]string)

		for _, cookie := range cookies {
			if strings.HasPrefix(cookie.Name, "flash_") {
				// Decode each flash message
				flashMessages[strings.TrimPrefix(cookie.Name, "flash_")] = cookie.Value

				// Delete the cookie
				expiredCookie := &http.Cookie{
					Name:   cookie.Name,
					Value:  "",
					Path:   "/",
					MaxAge: -1,
				}
				http.SetCookie(w, expiredCookie)
			}
		}

		// Store the messages in the context
		ctx := context.WithValue(r.Context(), flashMessagesKey, flashMessages)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func GetFlashMessages(ctx context.Context) map[string]string {
	if messages, ok := ctx.Value(flashMessagesKey).(map[string]string); ok {
		return messages
	}

	return nil
}

func GetFlashMessage(ctx context.Context, name string) (string, bool) {
	messages := GetFlashMessages(ctx)
	msg, ok := messages[name]

	res, err := url.QueryUnescape(msg)
	if err != nil {
		return "", false
	}

	return res, ok
}

func SetFlashMessage(w http.ResponseWriter, name string, message string) {
	cookie := &http.Cookie{
		Name:     "flash_" + name, // Unique name for multiple messages
		Value:    url.QueryEscape(message),
		Path:     "/",
		HttpOnly: true,
		MaxAge:   60,
	}
	http.SetCookie(w, cookie)
}
