package ui

import (
	"net/http"
	"os"

	"github.com/anthdm/superkit/kit"
	"github.com/go-chi/chi/v5"

	"github.com/godlew/homecoin/public"
)

func RegisterStatic(router chi.Router) {
	if kit.IsDevelopment() {
		router.Handle("/public/*", disableCache(staticDev()))
		return
	}
	if kit.IsProduction() {
		router.Handle("/public/*", staticProd())
		return
	}
	router.Handle("/public/*", staticDev())
}

func staticDev() http.Handler {
	return http.StripPrefix("/public/", http.FileServerFS(os.DirFS("public")))
}

func staticProd() http.Handler {
	return http.StripPrefix("/public/", http.FileServerFS(public.AssetsFS))
}

func disableCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
