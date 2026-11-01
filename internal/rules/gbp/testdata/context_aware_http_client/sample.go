package sample

import "net/http"

func f() {
	http.Get("http://example.com")                    // want "avoid http.Get"
	http.NewRequest("GET", "http://example.com", nil) // want "NewRequestWithContext"
	http.NewRequestWithContext(nil, "GET", "http://example.com", nil)
}
