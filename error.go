package smooch

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ResponseData defines data for every response
type ResponseData struct {
	HTTPCode int
	Flag     string
}

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

func checkSmoochError(r *http.Response) (*ResponseData, error) {
	var errorPayload ErrorPayload
	decodeErr := json.NewDecoder(r.Body).Decode(&errorPayload)
	if decodeErr != nil {
		return nil, decodeErr
	}

	err := &SmoochError{
		message: fmt.Sprintf("StatusCode: %d Code: %s Message: %s",
			r.StatusCode,
			errorPayload.Details.Code,
			errorPayload.Details.Description,
		),
		code: r.StatusCode,
	}

	respData := &ResponseData{
		HTTPCode: r.StatusCode,
		Flag:     errorPayload.Details.Code,
	}
	return respData, err
}
