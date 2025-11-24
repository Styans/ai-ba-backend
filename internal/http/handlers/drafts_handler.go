package handlers

import (
	"net/http"
)

func ListDraftsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("list drafts"))
}

func GetDraftHandler(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("get draft"))
}

func ReviewDraftHandler(w http.ResponseWriter, r *http.Request) {
	// TODO
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("review draft"))
}
