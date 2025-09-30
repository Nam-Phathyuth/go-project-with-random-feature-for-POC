package pkg

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type TaskError struct {
	Message string
	Err     error
}

func (e TaskError) Error() string {
	log.Println(e.Err.Error())
	return e.Message
}

type HttpError struct {
	Message string `json:"message"`
	Err     error  `json:"error"`
	Code    int    `json:"code"`
}

func HandleError(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				httpErr := HttpError{Message: "Internal Server Error", Err: nil, Code: http.StatusInternalServerError}
				log.Printf("error: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set("Content-Type", "Application/json")
				jsonData, _ := json.Marshal(httpErr)
				w.Write(jsonData)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

var (
	ErrNotFound = errors.New("not found")
)
