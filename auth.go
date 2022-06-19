package main

import (
	"net/http"
)

func middleWare(h func(w http.ResponseWriter, r *http.Request, u *session)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := r.Header["Auth-Token"]
		if !ok || len(token) == 0 {
			reportError(http.StatusBadRequest, w, errorToken)
			return
		}
		user, err := checkToken(token[0])
		if err != nil {
			reportError(http.StatusBadRequest, w, err)
			return
		}
		h(w, r, user)
	})
}
