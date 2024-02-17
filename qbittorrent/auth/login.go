package auth

import (
	"net/http"
)

func Login(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Ok."))
	w.WriteHeader(http.StatusOK)
}
