package smooch

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckSmoochError(t *testing.T) {
	errorJsonString := `
	{
		"error": {
			"code": "unauthorized",
			"description": "Authorization is required"
		}
	}`

	r := ioutil.NopCloser(bytes.NewReader([]byte(errorJsonString)))
	defer r.Close()

	response := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       r,
	}

	err := checkSmoochError(response)
	assert.Error(t, err)
	assert.EqualError(t, err, "StatusCode: 401 Code: unauthorized Message: Authorization is required")
}
