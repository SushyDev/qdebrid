package app

import (
	"net/http"
)

func (*Module) Version(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("2.0"))
}
