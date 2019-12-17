package smooch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/gomodule/redigo/redis"
	"github.com/kitabisa/smooch/storage"
)

var (
	ErrUserIDEmpty           = errors.New("user id is empty")
	ErrSurnameEmpty          = errors.New("surname is empty")
	ErrGivenNameEmpty        = errors.New("givenName is empty")
	ErrPhonenumberEmpty      = errors.New("phonenumber is empty")
	ErrChannelTypeEmpty      = errors.New("channel type is empty")
	ErrConfirmationTypeEmpty = errors.New("confirmation type is empty")
	ErrKeyIDEmpty            = errors.New("key id is empty")
	ErrSecretEmpty           = errors.New("secret is empty")
	ErrRedisNil              = errors.New("redis pool is nil")
	ErrMessageNil            = errors.New("message is nil")
	ErrMessageRoleEmpty      = errors.New("message.Role is empty")
	ErrMessageTypeEmpty      = errors.New("message.Type is empty")
	ErrDecodeToken           = errors.New("error decode token")
	ErrWrongAuth             = errors.New("error wrong authentication")
)

const (
	RegionUS = "US"
	RegionEU = "EU"

	usRootURL = "https://api.smooch.io"
	euRootURL = "https://api.eu-1.smooch.io"

	contentTypeHeaderKey   = "Content-Type"
	authorizationHeaderKey = "Authorization"

	contentTypeJSON = "application/json"
)

type Options struct {
	Auth       string
	AppID      string
	KeyID      string
	Secret     string
	WebhookURL string
	Mux        *http.ServeMux
	Logger     Logger
	Region     string
	HttpClient *http.Client
	RedisPool  *redis.Pool
}

type WebhookEventHandler func(payload *Payload)

type Client interface {
	Handler() http.Handler
	IsJWTExpired() (bool, error)
	RenewToken() (string, error)
	AddWebhookEventHandler(handler WebhookEventHandler)
	Send(userID string, message *Message) (*ResponsePayload, error)
	SendHSM(userID string, hsmMessage *HsmMessage) (*ResponsePayload, error)
	GetAppUser(userID string) (*AppUser, error)
	PreCreateAppUser(userID, surname, givenName string) (*AppUser, error)
	LinkAppUserToChannel(channelType, confirmationType, phoneNumber string) (*AppUser, error)
	UploadFileAttachment(filepath string, upload AttachmentUpload) (*Attachment, error)
	UploadAttachment(r io.Reader, upload AttachmentUpload) (*Attachment, error)
}

type SmoochClient struct {
	Mux                  *http.ServeMux
	Auth                 string
	AppID                string
	KeyID                string
	Secret               string
	Logger               Logger
	Region               string
	WebhookEventHandlers []WebhookEventHandler
	HttpClient           *http.Client
	Mtx                  sync.Mutex
	RedisStorage         *storage.RedisStorage
}

func New(o Options) (*SmoochClient, error) {
	if o.KeyID == "" {
		return nil, ErrKeyIDEmpty
	}

	if o.Secret == "" {
		return nil, ErrSecretEmpty
	}

	if o.Mux == nil {
		o.Mux = http.NewServeMux()
	}

	if o.HttpClient == nil {
		o.HttpClient = http.DefaultClient
	}

	if o.Region == "" {
		o.Region = RegionUS
	}

	if o.WebhookURL == "" {
		o.WebhookURL = "/"
	}

	if o.Logger == nil {
		o.Logger = &nopLogger{}
	}

	region := RegionUS
	if o.Region == "EU" {
		region = RegionEU
	}

	if o.Auth != AuthBasic && o.Auth != AuthJWT {
		return nil, ErrWrongAuth
	}

	sc := &SmoochClient{
		Auth:       o.Auth,
		Mux:        o.Mux,
		AppID:      o.AppID,
		KeyID:      o.KeyID,
		Secret:     o.Secret,
		Logger:     o.Logger,
		Region:     region,
		HttpClient: o.HttpClient,
	}

	if sc.Auth == AuthJWT {
		if o.RedisPool == nil {
			return nil, ErrRedisNil
		}

		sc.RedisStorage = storage.NewRedisStorage(o.RedisPool)

		_, err := sc.RedisStorage.GetTokenFromRedis()
		if err != nil {
			_, err := sc.RenewToken()
			if err != nil {
				return nil, err
			}
		}
	}

	sc.Mux.HandleFunc(o.WebhookURL, sc.handle)
	return sc, nil
}

func (sc *SmoochClient) Handler() http.Handler {
	return sc.Mux
}

// IsJWTExpired will check whether Smooch JWT is expired or not.
func (sc *SmoochClient) IsJWTExpired() (bool, error) {
	jwtToken, err := sc.RedisStorage.GetTokenFromRedis()
	if err != nil {
		if err == redis.ErrNil {
			return true, nil
		}
		return false, err
	}
	return isJWTExpired(jwtToken, sc.Secret)
}

// RenewToken will generate new Smooch JWT token.
func (sc *SmoochClient) RenewToken() (string, error) {
	sc.Mtx.Lock()
	defer sc.Mtx.Unlock()

	jwtToken, err := GenerateJWT("app", sc.KeyID, sc.Secret)
	if err != nil {
		return "", err
	}

	err = sc.RedisStorage.SaveTokenToRedis(jwtToken, JWTExpiration)
	if err != nil {
		return "", err
	}

	return jwtToken, nil
}

func (sc *SmoochClient) AddWebhookEventHandler(handler WebhookEventHandler) {
	sc.WebhookEventHandlers = append(sc.WebhookEventHandlers, handler)
}

func (sc *SmoochClient) Send(userID string, message *Message) (*ResponsePayload, *ResponseData, error) {
	if userID == "" {
		return nil, nil, ErrUserIDEmpty
	}

	if message == nil {
		return nil, nil, ErrMessageNil
	}

	if message.Role == "" {
		return nil, nil, ErrMessageRoleEmpty
	}

	if message.Type == "" {
		return nil, nil, ErrMessageTypeEmpty
	}

	url := sc.getURL(
		fmt.Sprintf("/v1.1/apps/%s/appusers/%s/messages", sc.AppID, userID),
		nil,
	)

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(message)
	if err != nil {
		return nil, nil, err
	}

	req, err := sc.createRequest(http.MethodPost, url, buf, nil)
	if err != nil {
		return nil, nil, err
	}

	var responsePayload ResponsePayload
	respData, err := sc.sendRequest(req, &responsePayload)
	if err != nil {
		return nil, respData, err
	}

	return &responsePayload, respData, nil
}

// SendHSM will send message using Whatsapp HSM template
func (sc *SmoochClient) SendHSM(userID string, hsmMessage *HsmMessage) (*ResponsePayload, *ResponseData, error) {
	url := sc.getURL(
		fmt.Sprintf("/v1.1/apps/%s/appusers/%s/messages", sc.AppID, userID),
		nil,
	)

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(hsmMessage)
	if err != nil {
		return nil, nil, err
	}

	req, err := sc.createRequest(http.MethodPost, url, buf, nil)
	if err != nil {
		return nil, nil, err
	}

	var responsePayload ResponsePayload
	respData, err := sc.sendRequest(req, &responsePayload)
	if err != nil {
		return nil, respData, err
	}

	return &responsePayload, respData, nil
}

func (sc *SmoochClient) GetAppUser(userID string) (*AppUser, *ResponseData, error) {
	url := sc.getURL(
		fmt.Sprintf("/v1.1/apps/%s/appusers/%s", sc.AppID, userID),
		nil,
	)

	req, err := sc.createRequest(http.MethodGet, url, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	var response GetAppUserResponse
	respData, err := sc.sendRequest(req, &response)
	if err != nil {
		return nil, respData, err
	}

	return response.AppUser, respData, nil
}

// PreCreateAppUser will register user to smooch
func (sc *SmoochClient) PreCreateAppUser(userID, surname, givenName string) (*AppUser, *ResponseData, error) {
	url := sc.getURL(
		fmt.Sprintf("/v1.1/apps/%s/appusers", sc.AppID),
		nil,
	)

	if userID == "" {
		return nil, nil, ErrUserIDEmpty
	}

	if surname == "" {
		return nil, nil, ErrSurnameEmpty
	}

	if givenName == "" {
		return nil, nil, ErrGivenNameEmpty
	}

	payload := PreCreateAppUserPayload{
		UserID:    userID,
		Surname:   surname,
		GivenName: givenName,
	}

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(payload)
	if err != nil {
		return nil, nil, err
	}

	req, err := sc.createRequest(http.MethodPost, url, buf, nil)
	if err != nil {
		return nil, nil, err
	}

	var response PreCreateAppUserResponse
	respData, err := sc.sendRequest(req, &response)
	if err != nil {
		return nil, respData, err
	}

	return response.AppUser, respData, nil
}

// LinkAppUserToChannel will link user to specifiied channel
func (sc *SmoochClient) LinkAppUserToChannel(userID, channelType, confirmationType, phoneNumber string) (*AppUser, *ResponseData, error) {
	url := sc.getURL(
		fmt.Sprintf("/v1.1/apps/%s/appusers/%s/channels", sc.AppID, userID),
		nil,
	)

	if userID == "" {
		return nil, nil, ErrUserIDEmpty
	}

	if channelType == "" {
		return nil, nil, ErrChannelTypeEmpty
	}

	if confirmationType == "" {
		return nil, nil, ErrConfirmationTypeEmpty
	}

	if phoneNumber == "" {
		return nil, nil, ErrPhonenumberEmpty
	}

	payload := LinkAppUserToChannelPayload{
		Type: channelType,
		Confirmation: LinkAppConfirmationData{
			Type: confirmationType,
		},
		PhoneNumber: phoneNumber,
	}

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(payload)
	if err != nil {
		return nil, nil, err
	}

	req, err := sc.createRequest(http.MethodPost, url, buf, nil)
	if err != nil {
		return nil, nil, err
	}

	var response LinkAppUserToChannelResponse
	respData, err := sc.sendRequest(req, &response)
	if err != nil {
		return nil, respData, err
	}

	return response.AppUser, respData, nil
}

func (sc *SmoochClient) UploadFileAttachment(filepath string, upload AttachmentUpload) (*Attachment, *ResponseData, error) {
	r, err := os.Open(filepath)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	return sc.UploadAttachment(r, upload)

}
func (sc *SmoochClient) UploadAttachment(r io.Reader, upload AttachmentUpload) (*Attachment, *ResponseData, error) {

	queryParams := url.Values{
		"access": []string{upload.Access},
	}
	if upload.For != "" {
		queryParams["for"] = []string{upload.For}
	}
	if upload.AppUserID != "" {
		queryParams["appUserId"] = []string{upload.AppUserID}
	}
	if upload.UserID != "" {
		queryParams["userId"] = []string{upload.UserID}
	}

	url := sc.getURL(
		fmt.Sprintf("/v1.1/apps/%s/attachments", sc.AppID),
		queryParams,
	)

	formData := map[string]io.Reader{
		"source": r,
		"type":   strings.NewReader(upload.MIMEType),
	}

	req, err := sc.createMultipartRequest(url, formData)
	if err != nil {
		return nil, nil, err
	}

	var response Attachment
	respData, err := sc.sendRequest(req, &response)
	if err != nil {
		return nil, respData, err
	}

	return &response, respData, nil
}

func (sc *SmoochClient) DeleteAttachment(attachment *Attachment) (*ResponseData, error) {
	url := sc.getURL(
		fmt.Sprintf("/v1.1/apps/%s/attachments", sc.AppID),
		nil,
	)

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(attachment)
	if err != nil {
		return nil, err
	}

	req, err := sc.createRequest(http.MethodPost, url, buf, nil)
	if err != nil {
		return nil, err
	}

	respData, err := sc.sendRequest(req, nil)
	if err != nil {
		return respData, err
	}

	return respData, nil
}

func (sc *SmoochClient) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		sc.Logger.Errorw("request body read failed", "err", err)
		return
	}

	var payload Payload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		sc.Logger.Errorw("could not decode response", "err", err)
		return
	}

	w.WriteHeader(http.StatusOK)

	sc.dispatch(&payload)
}

func (sc *SmoochClient) dispatch(p *Payload) {
	for _, handler := range sc.WebhookEventHandlers {
		handler(p)
	}
}

func (sc *SmoochClient) getURL(endpoint string, values url.Values) string {
	rootURL := usRootURL
	if sc.Region == RegionEU {
		rootURL = euRootURL
	}

	u, err := url.Parse(rootURL)
	if err != nil {
		panic(err)
	}

	u.Path = path.Join(u.Path, endpoint)
	if len(values) > 0 {
		u.RawQuery = values.Encode()
	}
	return u.String()
}

func (sc *SmoochClient) createRequest(
	method string,
	url string,
	buf *bytes.Buffer,
	header http.Header) (*http.Request, error) {

	var req *http.Request
	var err error
	var jwtToken string

	if header == nil {
		header = http.Header{}
	}

	if header.Get(contentTypeHeaderKey) == "" {
		header.Set(contentTypeHeaderKey, contentTypeJSON)
	}

	if sc.Auth == AuthJWT {
		isExpired, err := sc.IsJWTExpired()
		if err != nil {
			return nil, err
		}

		if isExpired {
			jwtToken, err = sc.RenewToken()
			if err != nil {
				return nil, err
			}
		} else {
			jwtToken, err = sc.RedisStorage.GetTokenFromRedis()
			if err != nil {
				return nil, err
			}
		}

		header.Set(authorizationHeaderKey, fmt.Sprintf("Bearer %s", jwtToken))
	}

	if buf == nil {
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, buf)
	}

	if err != nil {
		return nil, err
	}
	req.Header = header

	if sc.Auth == AuthBasic {
		req.SetBasicAuth(sc.KeyID, sc.Secret)
	}

	return req, nil
}

func (sc *SmoochClient) createMultipartRequest(
	url string,
	values map[string]io.Reader) (*http.Request, error) {
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	var err error

	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// Add an image file
		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				return nil, err
			}
		} else if fbr, ok := r.(*BytesFileReader); ok {
			if fw, err = w.CreateFormFile(key, fbr.Filename); err != nil {
				return nil, err
			}
		} else {
			// Add other fields
			if fw, err = w.CreateFormField(key); err != nil {
				return nil, err
			}
		}

		if _, err = io.Copy(fw, r); err != nil {
			return nil, err
		}
	}
	w.Close()

	header := http.Header{}
	header.Set("Content-Type", w.FormDataContentType())

	req, err := sc.createRequest(http.MethodPost, url, buf, header)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (sc *SmoochClient) sendRequest(req *http.Request, v interface{}) (*ResponseData, error) {
	response, err := sc.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		if v != nil {
			err := json.NewDecoder(response.Body).Decode(&v)
			if err != nil {
				return nil, err
			}
		}

		respData := &ResponseData{
			HTTPCode: response.StatusCode,
		}
		return respData, nil
	}
	return checkSmoochError(response)
}
