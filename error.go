package smooch

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type SmoochError struct {
	message string
	code    int
}

func (e *SmoochError) Code() int {
	return e.code
}

func (e *SmoochError) Error() string {
	return e.message
}

func checkSmoochError(r *http.Response) error {
	var errorPayload ErrorPayload
	decodeErr := json.NewDecoder(r.Body).Decode(&errorPayload)
	if decodeErr != nil {
		return decodeErr
	}

	err := &SmoochError{
		message: fmt.Sprintf("StatusCode: %d Code: %s Message: %s",
			r.StatusCode,
			errorPayload.Details.Code,
			errorPayload.Details.Description,
		),
		code: r.StatusCode,
	}

	return err
}
