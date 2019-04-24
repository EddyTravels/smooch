package smooch

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	payloadExample1 = `
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
			"conversationStarted": true
		},
		"conversation": {
			"_id": "105e47578be874292d365ee8"
		},
		"version": "v1.1"
	}
	`

	errorPayloadExample = `
	{
		"trigger": "message:delivery:failure",
		"app": {
			"_id": "575040549a38df8fb4eb1e51"
		},
		"appUser": {
			"_id": "de13bee15b51033b34162411",
			"userId": "123",
			"surname": "Steve",
			"givenName": "Bob",
			"signedUpAt": "2018-04-02T14:45:46.505Z",
			"properties": { "favoriteFood": "prizza" }
		},
		"conversation": {
			"_id": "105e47578be874292d365ee8"
		},
		"destination": {
			"type": "line"
		},
		"isFinalEvent": true,
		"error": {
			"code": "unauthorized",
			"underlyingError": {
				"message":
					"Authentication failed due to the following reason: invalid token. Confirm that the access token in the authorization header is valid."
			}
		},
		"message": {
			"_id": "5baa610db5bebb000ce855d6"
		},
		"timestamp": 1480001711.941,
		"version": "v1.1"
	}`
)

func TestPayloadDecode(t *testing.T) {
	payload := &Payload{}
	err := json.Unmarshal([]byte(payloadExample1), &payload)
	assert.NoError(t, err)
	assert.Len(t, payload.Messages, 1)

	assert.Equal(t, payload.Trigger, "message:appUser")
	assert.Equal(t, payload.App.ID, "5698edbf2a43bd081be982f1")
	assert.Equal(t, payload.Messages[0].ID, "55c8c1498590aa1900b9b9b1")
	assert.Equal(t, payload.Messages[0].Type, MessageTypeText)
	assert.Equal(t, payload.Messages[0].Text, "Hi! Do you have time to chat?")
	assert.Equal(t, payload.Messages[0].Role, RoleAppUser)
	assert.Equal(t, payload.Messages[0].AuthorID, "c7f6e6d6c3a637261bd9656f")
	assert.Equal(t, payload.Messages[0].Name, "Steve")
	assert.Equal(t, payload.Messages[0].Received, time.Unix(1444348338, 704*nsMultiplier))
	assert.Equal(t, payload.Messages[0].Source.Type, SourceTypeMessenger)
	assert.Equal(t, payload.AppUser.ID, "c7f6e6d6c3a637261bd9656f")
	assert.Equal(t, payload.AppUser.UserID, "bob@example.com")
	assert.Equal(t, payload.AppUser.ConversationStarted, true)
	assert.Equal(t, payload.Conversation.ID, "105e47578be874292d365ee8")
	assert.Equal(t, payload.Version, "v1.1")
}

func TestPayloadEncode(t *testing.T) {
	p := &Payload{}
	err := json.Unmarshal([]byte(payloadExample1), &p)
	assert.NoError(t, err)
	assert.Len(t, p.Messages, 1)
	p.Messages[0].Received = time.Unix(1444348340, 420*nsMultiplier)

	data, err := json.Marshal(p)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	payload := &Payload{}
	err = json.Unmarshal(data, &payload)
	assert.NoError(t, err)
	assert.Len(t, payload.Messages, 1)
	assert.Equal(t, payload.Messages[0].Received, time.Unix(1444348340, 420*nsMultiplier))
}

func TestErrorPayloadDecode(t *testing.T) {
	payload := &Payload{}
	err := json.Unmarshal([]byte(errorPayloadExample), &payload)
	assert.NoError(t, err)
	assert.Equal(t, TriggerMessageDeliveryFailure, payload.Trigger)
	assert.Equal(t, "575040549a38df8fb4eb1e51", payload.App.ID)
	assert.Equal(t, "de13bee15b51033b34162411", payload.AppUser.ID)
	assert.Equal(t, "123", payload.AppUser.UserID)
	assert.Equal(t, "Steve", payload.AppUser.Surname)
	assert.Equal(t, "Bob", payload.AppUser.GivenName)
	assert.Equal(t, "2018-04-02T14:45:46Z", payload.AppUser.SignedUpAt.Format(time.RFC3339))
	assert.Equal(t, "105e47578be874292d365ee8", payload.Conversation.ID)
	assert.Equal(t, true, payload.IsFinalEvent)
	assert.Equal(t, "unauthorized", payload.Error.Code)
	assert.Equal(t,
		"Authentication failed due to the following reason: invalid token. Confirm that the access token in the authorization header is valid.",
		payload.Error.UnderlyingError["message"],
	)

	assert.Equal(t, "5baa610db5bebb000ce855d6", payload.Message.ID)
}
