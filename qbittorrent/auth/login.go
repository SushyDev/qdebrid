package auth

import (
	"net/http"
)

func Login(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
