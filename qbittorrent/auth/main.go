package auth

import (
	"net/http"
)

// TODO: Maybe implement??

func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}
