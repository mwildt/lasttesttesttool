package utils

import (
	"errors"
	"fmt"
	"github.com/mwildt/load-monitor/pkg/session"
	"net/http"
	"time"
)

func DeleteCookie(name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		HttpOnly: true,
	}
}

func CreateSessionCookie[T any](name string, sess session.Session[T]) *http.Cookie {
	return CreateSessionCookieByKey(name, sess.Id)
}

func CreateSessionCookieByKey(name string, key string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    key,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		HttpOnly: true,
	}
}

func ReadSessionId(request *http.Request, sessionIdKey string) (res string, err error) {
	cookie, err := request.Cookie(sessionIdKey)
	if errors.Is(err, http.ErrNoCookie) || cookie.Value == "" { // keine session-id in request vorhanden
		return res, fmt.Errorf("unauthorized")
	} else if err != nil {
		return res, fmt.Errorf("unauthorized")
	} else {
		return cookie.Value, nil
	}
}
