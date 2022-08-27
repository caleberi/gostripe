package main

import "net/http"

func SessionLoader(next http.Handler) http.Handler {
	return session.LoadAndSave(next)
}
