package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
)

var (
	errUnsupportedContentType = errors.New("unsupported ContentType")
)

func writeContentTypeError(w http.ResponseWriter, ct, ctJSON string) bool {
	if err := validateContentType(ct, ctJSON); err != nil {
		if errors.Is(err, errUnsupportedContentType) {
			slog.Info(
				"unsupported Content-Type",
				"content_type", ct,
			)
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return true
		}
		slog.Info("unknown error", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return true
	}
	return false
}

func successfulDecoding(w http.ResponseWriter, r *http.Request, a any) bool {
	if err := decodeJSON(r.Body, a); err != nil {
		slog.Info("failed to decode data", "error", err)
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

func successfulEncoding(w http.ResponseWriter, a any) bool {
	if err := encodeJSON(w, a); err != nil {
		slog.Info("failed to encode data", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return false
	}
	return true
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set(contentType, contentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}

func encodeJSON(w http.ResponseWriter, a any) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(a)
}

func decodeJSON(b io.Reader, a any) error {
	dec := json.NewDecoder(b)
	dec.DisallowUnknownFields()

	return dec.Decode(a)
}

func validateContentType(contType, expContType string) error {
	mediaType, _, err := mime.ParseMediaType(contType)
	if err != nil || mediaType != expContType {
		return errUnsupportedContentType
	}
	return nil
}
