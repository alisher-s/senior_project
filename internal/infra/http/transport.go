package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// DecodeAndValidate decodes JSON body into dst and validates it using go-playground/validator.
// It disallows unknown fields to reduce accidental API misuse.
func DecodeAndValidate(r *http.Request, dst any, v *validator.Validate) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return v.Struct(dst)
}

