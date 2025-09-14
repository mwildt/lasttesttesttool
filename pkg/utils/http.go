package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

func SendStatus(w http.ResponseWriter, request *http.Request, code int) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
}

func ReadJsonBody[T any](request *http.Request) (res T, _ error) {
	err := json.NewDecoder(request.Body).Decode(&res)
	return res, err
}

func SendJson(w http.ResponseWriter, request *http.Request, code int, data interface{}) {
	payload, err := json.Marshal(data)
	if err != nil {
		InternalServerError(w, request, err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(code)
		w.Write(payload)
	}
}

func StatusForbidden(w http.ResponseWriter, request *http.Request) {
	SendStatus(w, request, http.StatusForbidden)
}

func StatusUnauthorized(w http.ResponseWriter, request *http.Request) {
	SendStatus(w, request, http.StatusUnauthorized)
}

func NotFound(w http.ResponseWriter, request *http.Request) {
	SendStatus(w, request, http.StatusNotFound)
}

func BadRequest(w http.ResponseWriter, request *http.Request) {
	SendStatus(w, request, http.StatusBadRequest)
}

func BadRequestJson(w http.ResponseWriter, request *http.Request, data interface{}) {
	SendJson(w, request, http.StatusBadRequest, data)
}

func InternalServerError(w http.ResponseWriter, request *http.Request, err error) {
	log.Println(err.Error())
	SendStatus(w, request, http.StatusInternalServerError)
}

func Ok(w http.ResponseWriter, request *http.Request) {
	SendStatus(w, request, http.StatusOK)
}

func OkJson(w http.ResponseWriter, request *http.Request, data interface{}) {
	SendJson(w, request, http.StatusOK, data)
}

func CreatedJson(w http.ResponseWriter, request *http.Request, data interface{}) {
	SendJson(w, request, http.StatusCreated, data)
}

type StatusLoggingResponseWriter struct {
	http.ResponseWriter
	Status int
}

func NewStatusLoggingResponseWriter(w http.ResponseWriter) *StatusLoggingResponseWriter {
	return &StatusLoggingResponseWriter{w, http.StatusOK}
}

func (lrw *StatusLoggingResponseWriter) WriteHeader(code int) {
	lrw.Status = code
	lrw.ResponseWriter.WriteHeader(code)
}
