package app

import (
	"net/http"
)

func Version(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("2.0"))
}
