package auth

import (
	"net/http"
)

func (*Module) Login(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Ok."))
	w.WriteHeader(http.StatusOK)
}
