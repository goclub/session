package sess

import (
	"net/http"
	"time"
)

func newCookieFromOption(value string, option HubOptionCookie) *http.Cookie {
	var expires time.Time
	if option.MaxAge > 0 {
		d := time.Duration(option.MaxAge) * time.Second
		expires = time.Now().Add(d)
	} else if option.MaxAge < 0 {
		// Set it to the past to expire now.
		expires = time.Unix(1, 0)
	}
	return &http.Cookie{
		Name:     option.Name,
		Value:    value,
		Path:     option.Path,
		Domain:   option.Domain,
		MaxAge:   option.MaxAge,
		Expires:  expires,
		Secure:   option.Secure,
		HttpOnly: true,
	}
}
