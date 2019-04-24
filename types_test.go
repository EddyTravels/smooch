package smooch

import (
	"encoding/json"
	"fmt"
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
)

func TestMessageDecode(t *testing.T) {
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

func TestMessageEncode(t *testing.T) {
	p := &Payload{}
	err := json.Unmarshal([]byte(payloadExample1), &p)
	assert.NoError(t, err)
	assert.Len(t, p.Messages, 1)
	p.Messages[0].Received = time.Unix(1444348340, 420*nsMultiplier)

	data, err := json.Marshal(p)
	assert.NoError(t, err)
	assert.NotNil(t, data)

	fmt.Println("JSON: ", string(data))

	payload := &Payload{}
	err = json.Unmarshal(data, &payload)
	assert.NoError(t, err)
	assert.Len(t, payload.Messages, 1)
	assert.Equal(t, payload.Messages[0].Received, time.Unix(1444348340, 420*nsMultiplier))
}
