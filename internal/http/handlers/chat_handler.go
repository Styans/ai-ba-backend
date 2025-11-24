package handlers

import (
	"net/http"
)

func StartChatHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: реализация
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("start chat"))
}

func SendChatHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: реализация
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("send chat"))
}
