package main

/*
import "net/http"

type authMiddleWare struct {
	tokens map[string]session
}

func (a *authMiddleWare) MiddleWare(h http.Handler) http.Handler {
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
	})
}
*/
//call the function within a function but with another attribute to get around the scope issue and to have less code
