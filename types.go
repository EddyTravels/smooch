package smooch

import (
	"bytes"
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
	SourceTypeWhatsApp  = "whatsapp"

	RoleAppUser  = Role("appUser")
	RoleAppMaker = Role("appMaker")

	TriggerMessageAppUser         = "message:appUser"
	TriggerMessageAppMaker        = "message:appMaker"
	TriggerMessageDeliveryFailure = "message:delivery:failure"
	TriggerMessageDeliveryChannel = "message:delivery:channel"
	TriggerMessageDeliveryUser    = "message:delivery:user"

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
	HasPaymentInfo      bool                   `json:"hasPaymentInfo,omitmepty"`
}

type AppUserClient struct {
	ID            string                 `json:"_id,omitempty"`
	Platform      string                 `json:"platform,omitempty"`
	IntegrationId string                 `json:"integrationId,omitempty"`
	Primary       bool                   `json:"primary"`
	Active        bool                   `json:"active"`
	DeviceID      string                 `json:"deviceId,omitempty"`
	DisplayName   string                 `json:"displayName,omitempty"`
	AvatarURL     string                 `json:"avatarUrl,omitempty"`
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
	Actions     []*Action `json:"actions"`
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
	MediaURL        string                 `json:"mediaUrl,omitempty"`
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

// HsmLanguage defines hsm language payload
type HsmLanguage struct {
	Policy string `json:"policy"`
	Code   string `json:"code"`
}

// HsmLocalizableParams defines hsm localizable params data
type HsmLocalizableParams struct {
	Default interface{} `json:"default"`
}

// HsmPayload defines payload for hsm
type HsmPayload struct {
	Namespace         string                 `json:"namespace"`
	ElementName       string                 `json:"element_name"`
	Language          HsmLanguage            `json:"language"`
	LocalizableParams []HsmLocalizableParams `json:"localizable_params"`
}

// HsmMessageBody defines property for HSM message
type HsmMessageBody struct {
	Type MessageType `json:"type"`
	Hsm  HsmPayload  `json:"hsm"`
}

// HsmMessage defines struct payload for Whatsapp HSM message
type HsmMessage struct {
	Role          Role           `json:"role"`
	MessageSchema string         `json:"messageSchema"`
	Message       HsmMessageBody `json:"message"`
	Received      time.Time      `json:"received,omitempty"`
}

// UnmarshalJSON will unmarshall whatsapp HSM message
func (hm *HsmMessage) UnmarshalJSON(data []byte) error {
	type Alias HsmMessage
	aux := &struct {
		Received float64 `json:"received"`
		*Alias
	}{
		Alias: (*Alias)(hm),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	seconds := int64(aux.Received)
	ns := (int64(aux.Received*1000) - seconds*1000) * nsMultiplier
	hm.Received = time.Unix(seconds, ns)
	return nil
}

// MarshalJSON will marshall whatsapp HSM message
func (hm *HsmMessage) MarshalJSON() ([]byte, error) {
	type Alias HsmMessage
	aux := &struct {
		Received float64 `json:"received"`
		*Alias
	}{
		Alias: (*Alias)(hm),
	}
	aux.Received = float64(hm.Received.UnixNano()) / nsMultiplier
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

type AttachmentUpload struct {
	MIMEType string
	Access   string

	// optionals
	For       string
	AppUserID string
	UserID    string
}

func NewAttachmentUpload(mime string) AttachmentUpload {
	return AttachmentUpload{
		MIMEType: mime,
		Access:   "public",
	}
}

type Attachment struct {
	MediaURL  string `json:"mediaUrl"`
	MediaType string `json:"mediaType,omitempty"`
}

type BytesFileReader struct {
	*bytes.Reader
	Filename string
}

func NewBytesFileReader(filename string, b []byte) *BytesFileReader {
	r := bytes.NewReader(b)
	return &BytesFileReader{
		Reader:   r,
		Filename: filename,
	}
}
