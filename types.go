package smooch

import (
	"encoding/json"
	"time"
)

const (
	nsMultiplier = 1e9
)

const (
	MessageTypeText     = MessageType("text")
	MessageTypeImage    = MessageType("image")
	MessageTypeFile     = MessageType("file")
	MessageTypeLocation = MessageType("location")
	MessageTypeCarousel = MessageType("carousel")
	MessageTypeList     = MessageType("list")

	ActionTypePostback        = ActionType("postback")
	ActionTypeReply           = ActionType("reply")
	ActionTypeLocationRequest = ActionType("locationRequest")
	ActionTypeShare           = ActionType("share")
	ActionTypeBuy             = ActionType("buy")
	ActionTypeLink            = ActionType("link")
	ActionTypeWebview         = ActionType("webview")

	SourceTypeWeb       = "web"
	SourceTypeIOS       = "ios"
	SourceTypeAndroid   = "android"
	SourceTypeMessenger = "messenger"
	SourceTypeViber     = "viber"
	SourceTypeTelegram  = "telegram"
	SourceTypeWeChat    = "wechat"
	SourceTypeLine      = "line"
	SourceTypeTwilio    = "twilio"
	SourceTypeApi       = "api"

	RoleAppUser  = Role("appUser")
	RoleAppMaker = Role("appMaker")

	TriggerMessageAppUser         = "message:appUser"
	TriggerMessageAppMaker        = "message:appMaker"
	TriggerMessageDeliveryFailure = "message:delivery:failure"

	ImageRatioHorizontal = ImageRatio("horizontal")
	ImageRatioSquare     = ImageRatio("square")

	SizeCompact = Size("compact")
	SizeLarge   = Size("large")
)

type Role string

type MessageType string

type ActionType string

type Size string

type Payload struct {
	Trigger      string             `json:"trigger,omitempty"`
	App          Application        `json:"app,omitempty"`
	Messages     []*Message         `json:"messages,omitempty"`
	AppUser      AppUser            `json:"appUser,omitempty"`
	Conversation Conversation       `json:"conversation,omitempty"`
	Destination  *SourceDestination `json:"destination,omitempty"`
	IsFinalEvent bool               `json:"isFinalEvent"`
	Message      *TruncatedMessage  `json:"message,omitempty"`
	Error        *Error             `json:"error,omitempty"`
	Version      string             `json:"version,omitempty"`
}

type TruncatedMessage struct {
	ID string `json:"_id"`
}

type Error struct {
	Code            string                 `json:"code"`
	UnderlyingError map[string]interface{} `json:"underlyingError"`
	Message         string                 `json:"message"`
}

type Application struct {
	ID string `json:"_id,omitempty"`
}

type SourceDestination struct {
	Type              string `json:"type,omitempty"`
	Id                string `json:"id,omitempty"`
	IntegrationId     string `json:"integrationId,omitempty"`
	OriginalMessageId string `json:"originalMessageId,omitempty"`
}

type AppUser struct {
	ID                  string                 `json:"_id,omitempty"`
	UserID              string                 `json:"userId,omitempty"`
	Properties          map[string]interface{} `json:"properties,omitempty"`
	SignedUpAt          time.Time              `json:"signedUpAt,omitempty"`
	Clients             []*AppUserClient       `json:"clients,omitempty"`
	PendingClients      []*AppUserClient       `json:"pendingClients,omitempty"`
	ConversationStarted bool                   `json:"conversationStarted"`
	Email               string                 `json:"email,omitempty"`
	GivenName           string                 `json:"givenName,omitempty"`
	Surname             string                 `json:"surname,omitempty"`
}

type AppUserClient struct {
	ID            string                 `json:"_id,omitempty"`
	Platform      string                 `json:"platform,omitempty"`
	IntegrationId string                 `json:"integrationId,omitempty"`
	Primary       bool                   `json:"primary"`
	Active        bool                   `json:"active"`
	DeviceID      string                 `json:"deviceId,omitempty"`
	DisplayName   string                 `json:"displayName,omitempty"`
	AvatarUrl     string                 `json:"avatarUrl,omitempty"`
	Info          map[string]interface{} `json:"info,omitempty"`
	Raw           map[string]interface{} `json:"raw,omitempty"`
	AppVersion    string                 `json:"appVersion,omitempty"`
	LastSeen      time.Time              `json:"lastSeen,omitempty"`
	LinkedAt      time.Time              `json:"linkedAt,omitempty"`
	Blocked       bool                   `json:"blocked"`
}

type Conversation struct {
	ID          string `json:"_id"`
	UnreadCount int    `json:"unreadCount,omitempty"`
}

type Action struct {
	ID       string                 `json:"_id,omitempty"`
	Type     ActionType             `json:"type,omitempty"`
	Text     string                 `json:"text,omitempty"`
	Default  bool                   `json:"default,omitempty"`
	Payload  string                 `json:"payload,omitempty"`
	URI      string                 `json:"uri,omitempty"`
	Amount   int                    `json:"amount,omitempty"`
	Currency string                 `json:"currency,omitempty"`
	State    string                 `json:"state,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Item struct {
	ID          string    `json:"_id,omitempty"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Size        Size      `json:"size,omitempty"`
	MediaURL    string    `json:"mediaUrl,omitempty"`
	MediaType   string    `json:"mediaType,omitempty"`
	Actions     []*Action `json:"actions,omitempty"`
}

type Message struct {
	ID              string                 `json:"_id,omitempty"`
	Type            MessageType            `json:"type"`
	Text            string                 `json:"text,omitempty"`
	Role            Role                   `json:"role"`
	AuthorID        string                 `json:"authorId,omitempty"`
	Name            string                 `json:"name,omitempty"`
	Received        time.Time              `json:"received,omitempty"`
	Source          *SourceDestination     `json:"source,omitempty"`
	MediaUrl        string                 `json:"mediaUrl,omitempty"`
	Actions         []*Action              `json:"actions,omitempty"`
	Items           []*Item                `json:"items,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	DisplaySettings *DisplaySettings       `json:"displaySettings,omitempty"`
}

func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	aux := &struct {
		Received float64 `json:"received"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	seconds := int64(aux.Received)
	ns := (int64(aux.Received*1000) - seconds*1000) * nsMultiplier
	m.Received = time.Unix(seconds, ns)
	return nil
}

func (m *Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	aux := &struct {
		Received float64 `json:"received"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	aux.Received = float64(m.Received.UnixNano()) / nsMultiplier
	return json.Marshal(aux)
}

type MenuPayload struct {
	Menu Menu `json:"menu"`
}

type Menu struct {
	Items []*MenuItem `json:"items"`
}

type MenuItem struct {
	ID   string `json:"_id,omitempty"`
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
	URI  string `json:"uri,omitempty"`
}

type ImageRatio string

type DisplaySettings struct {
	ImageAspectRatio ImageRatio `json:"imageAspectRatio,omitempty"`
}

type ErrorPayload struct {
	Details ErrorDetails `json:"error"`
}

type ErrorDetails struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type ResponsePayload struct {
	Message       *Message      `json:"message,omitempty"`
	ExtraMessages []*Message    `json:"extraMessages,omitempty"`
	Conversation  *Conversation `json:"conversation,omitempty"`
}

type GetAppUserResponse struct {
	AppUser *AppUser `json:"appUser,omitempty"`
}
