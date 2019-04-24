package smooch

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	sampleResponse = `
	{
		"message": {
			"_id": "55c8c1498590aa1900b9b9b1",
			"authorId": "c7f6e6d6c3a637261bd9656f",
			"role": "appMaker",
			"type": "text",
			"name": "Steve",
			"text": "Just put some vinegar on it",
			"avatarUrl": "https://www.gravatar.com/image.jpg",
			"received": 1439220041.586
		},
		"extraMessages": [
			{
				"_id": "507f1f77bcf86cd799439011",
				"authorId": "c7f6e6d6c3a637261bd9656f",
				"role": "appMaker",
				"type": "image",
				"name": "Steve",
				"text": "Check out this image!",
				"mediaUrl": "http://example.org/image.jpg",
				"actions": [
					{
						"text": "More info",
						"type": "link",
						"uri": "http://example.org"
					}
				]
			}
		],
		"conversation": {
			"_id": "df0ebe56cbeab98589b8bfa7",
			"unreadCount": 0
		}
	}`

	sampleWebhookData = `
	{
		"trigger": "message:appUser",
		"app": {
			"_id": "5698edbf2a43bd081be982f1"
		},
		"messages": [
			{
				"_id": "55c8c1498590aa1900b9b9b1",
				"type": "text",
				"text": "Hi! Do you have time to chat?",
				"role": "appUser",
				"authorId": "c7f6e6d6c3a637261bd9656f",
				"name": "Steve",
				"received": 1444348338.704,
				"source": {
					"type": "messenger"
				}
			}
		],
		"appUser": {
			"_id": "c7f6e6d6c3a637261bd9656f",
			"userId": "bob@example.com",
			"conversationStarted": true,
			"surname": "Steve",
			"givenName": "Bob",
			"signedUpAt": "2018-04-02T14:45:46.505Z",
			"properties": { "favoriteFood": "prizza" }
		},
		"client": {
			"_id": "5c9d2f34a1d3a2504bc89511",
			"lastSeen": "2019-04-05T18:23:20.791Z",
			"platform": "web",
			"id": "20b2be30cf7e4152865f066930cbb5b2",
			"info": {
				"currentTitle": "Conversation Demo",
				"currentUrl": "https://examples.com/awesomechat",
				"browserLanguage": "en-US",
				"referrer": "",
				"userAgent":
					"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.119 Safari/537.36",
				"URL": "examples.com",
				"sdkVersion": "4.17.12",
				"vendor": "smooch"
			},
			"raw": {
				"currentTitle": "Conversation Demo",
				"currentUrl": "https://examples.com/awesomechat",
				"browserLanguage": "en-US",
				"referrer": "",
				"userAgent":
					"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.119 Safari/537.36",
				"URL": "examples.com",
				"sdkVersion": "4.17.12",
				"vendor": "smooch"
			},
			"active": true,
			"primary": true,
			"integrationId": "5c3640f8cd3fa5850931a954"
		},
		"conversation": {
			"_id": "105e47578be874292d365ee8"
		},
		"version": "v1.1"
	}`
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestGenerateJWT(t *testing.T) {
	secret := "a random, long, sequence of characters"
	token, err := GenerateJWT("app", "vienas", secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t,
		"eyJhbGciOiJIUzI1NiIsImtpZCI6InZpZW5hcyIsInR5cCI6IkpXVCJ9.eyJzY29wZSI6ImFwcCJ9.LDWhsxgx-E6zcPQr3Am2eD0nsTU6mD-ogRirbB2Pkdc",
		token,
	)
}

func TestSendOKResponse(t *testing.T) {
	fn := func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(sampleResponse))),
		}
	}

	sc, err := New(Options{
		VerifySecret: "very-secure-test-secret",
		HttpClient:   NewTestClient(fn),
	})
	assert.NoError(t, err)

	message := &Message{}
	response, err := sc.Send("", message)
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.EqualError(t, err, ErrUserIDEmpty.Error())

	response, err = sc.Send("TestUser", nil)
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.EqualError(t, err, ErrMessageNil.Error())

	response, err = sc.Send("TestUser", message)
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.EqualError(t, err, ErrMessageRoleEmpty.Error())

	message = &Message{
		Role: RoleAppUser,
	}
	response, err = sc.Send("TestUser", message)
	assert.Nil(t, response)
	assert.Error(t, err)
	assert.EqualError(t, err, ErrMessageTypeEmpty.Error())

	message = &Message{
		Role: RoleAppUser,
		Type: MessageTypeText,
	}
	response, err = sc.Send("TestUser", message)
	assert.NotNil(t, response)
	assert.NoError(t, err)

	assert.NotNil(t, response.Message)
	assert.Equal(t, "55c8c1498590aa1900b9b9b1", response.Message.ID)
	assert.Equal(t, 1, len(response.ExtraMessages))
	assert.Equal(t, "507f1f77bcf86cd799439011", response.ExtraMessages[0].ID)
	assert.Equal(t, "df0ebe56cbeab98589b8bfa7", response.Conversation.ID)
	assert.Equal(t, 0, response.Conversation.UnreadCount)
}

func TestSendErrorResponse(t *testing.T) {
	errorResponseJson := `
	{
		"error": {
			"code": "unauthorized",
			"description": "Authorization is required"
		}
	}`

	fn := func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(errorResponseJson))),
		}
	}

	sc, err := New(Options{
		VerifySecret: "very-secure-test-secret",
		HttpClient:   NewTestClient(fn),
	})
	assert.NoError(t, err)

	message := &Message{
		Role: RoleAppUser,
		Type: MessageTypeText,
	}
	response, err := sc.Send("TestUser", message)
	assert.Nil(t, response)
	assert.Error(t, err)

	smoochErr := err.(*SmoochError)
	assert.Equal(t, http.StatusUnauthorized, smoochErr.Code())
	assert.Equal(t,
		"StatusCode: 401 Code: unauthorized Message: Authorization is required",
		smoochErr.Error(),
	)
}

func TestHandlerOK(t *testing.T) {
	sc, err := New(Options{
		VerifySecret: "very-secure-test-secret",
	})
	assert.NoError(t, err)

	handlerInvokeCounter := 0

	sc.AddWebhookEventHandler(func(payload *Payload) {
		handlerInvokeCounter++
	})

	sc.AddWebhookEventHandler(func(payload *Payload) {
		handlerInvokeCounter++
	})

	mockData := bytes.NewReader([]byte(sampleWebhookData))
	req := httptest.NewRequest(http.MethodPost, "http://example.com/foo", mockData)
	req.Header.Set("X-Api-Key", "very-secure-test-secret")
	w := httptest.NewRecorder()

	handler := sc.Handler()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, "", string(body))
	assert.Equal(t, 2, handlerInvokeCounter)
}

func TestVerifyRequest(t *testing.T) {
	sc, err := New(Options{
		VerifySecret: "very-secure-test-secret",
	})
	r := &http.Request{}
	assert.NoError(t, err)
	assert.False(t, sc.VerifyRequest(r))

	r = &http.Request{
		Header: http.Header{},
	}
	r.Header.Set("X-Api-Key", "very-secure-test-secret")
	assert.NoError(t, err)
	assert.True(t, sc.VerifyRequest(r))

	sc, err = New(Options{
		VerifySecret: "very-secure-test-secret",
	})
	assert.NoError(t, err)

	headers := http.Header{}
	headers.Set("X-Api-Key", "very-secure-test-secret-wrong")
	r = &http.Request{
		Header: headers,
	}
	assert.NoError(t, err)
	assert.False(t, sc.VerifyRequest(r))

	headers = http.Header{}
	headers.Set("X-Api-Key", "very-secure-test-secret")
	r = &http.Request{
		Header: headers,
	}
	assert.NoError(t, err)
	assert.True(t, sc.VerifyRequest(r))
}
