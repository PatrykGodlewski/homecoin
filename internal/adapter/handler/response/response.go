package response

import (
	"encoding/json"
	"net/http"

	domainerrors "github.com/godlew/homecoin/internal/domain/errors"
)

type ErrorBody struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func Error(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	msg := "internal server error"

	switch err {
	case domainerrors.ErrNotFound:
		status = http.StatusNotFound
		msg = err.Error()
	case domainerrors.ErrAlreadyExists:
		status = http.StatusConflict
		msg = err.Error()
	case domainerrors.ErrUnauthorized:
		status = http.StatusUnauthorized
		msg = err.Error()
	case domainerrors.ErrForbidden:
		status = http.StatusForbidden
		msg = err.Error()
	case domainerrors.ErrAlreadyInHousehold:
		status = http.StatusConflict
		msg = err.Error()
	case domainerrors.ErrInvalidInput, domainerrors.ErrInvalidSplit:
		status = http.StatusBadRequest
		msg = err.Error()
	default:
		if err != nil {
			msg = err.Error()
		}
	}

	JSON(w, status, ErrorBody{Error: msg})
}
