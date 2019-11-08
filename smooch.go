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
	ErrUserIDEmpty       = errors.New("user id is empty")
	ErrKeyIDEmpty        = errors.New("key id is empty")
	ErrSecretEmpty       = errors.New("secret is empty")
	ErrRedisNil          = errors.New("redis pool is nil")
	ErrMessageNil        = errors.New("message is nil")
	ErrMessageRoleEmpty  = errors.New("message.Role is empty")
	ErrMessageTypeEmpty  = errors.New("message.Type is empty")
	ErrVerifySecretEmpty = errors.New("verify secret is empty")
	ErrDecodeToken       = errors.New("error decode token")
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
	AppID        string
	KeyID        string
	Secret       string
	VerifySecret string
	WebhookURL   string
	Mux          *http.ServeMux
	Logger       Logger
	Region       string
	HttpClient   *http.Client
	RedisPool    *redis.Pool
}

type WebhookEventHandler func(payload *Payload)

type Client interface {
	Handler() http.Handler
	IsJWTExpired() (bool, error)
	RenewToken() (string, error)
	AddWebhookEventHandler(handler WebhookEventHandler)
	Send(userID string, message *Message) (*ResponsePayload, error)
	VerifyRequest(r *http.Request) bool
	GetAppUser(userID string) (*AppUser, error)
	UploadFileAttachment(filepath string, upload AttachmentUpload) (*Attachment, error)
	UploadAttachment(r io.Reader, upload AttachmentUpload) (*Attachment, error)
}

type smoochClient struct {
	mux                  *http.ServeMux
	appID                string
	keyID                string
	secret               string
	verifySecret         string
	logger               Logger
	region               string
	webhookEventHandlers []WebhookEventHandler
	httpClient           *http.Client
	mtx                  sync.Mutex
	RedisStorage         *storage.RedisStorage
}

func New(o Options) (*smoochClient, error) {
	if o.KeyID == "" {
		return nil, ErrKeyIDEmpty
	}

	if o.Secret == "" {
		return nil, ErrSecretEmpty
	}

	if o.VerifySecret == "" {
		return nil, ErrVerifySecretEmpty
	}

	if o.RedisPool == nil {
		return nil, ErrRedisNil
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

	sc := &smoochClient{
		mux:          o.Mux,
		appID:        o.AppID,
		keyID:        o.KeyID,
		secret:       o.Secret,
		verifySecret: o.VerifySecret,
		logger:       o.Logger,
		region:       region,
		httpClient:   o.HttpClient,
		RedisStorage: storage.NewRedisStorage(o.RedisPool),
	}

	jwtToken, err := GenerateJWT("app", o.KeyID, o.Secret)
	if err != nil {
		return nil, err
	}

	// save token to redis
	err = sc.RedisStorage.SaveTokenToRedis(jwtToken, JWTExpiration)
	if err != nil {
		return nil, err
	}

	sc.mux.HandleFunc(o.WebhookURL, sc.handle)
	return sc, nil
}

func (sc *smoochClient) Handler() http.Handler {
	return sc.mux
}

// IsJWTExpired will check whether Smooch JWT is expired or not.
func (sc *smoochClient) IsJWTExpired() (bool, error) {
	jwtToken, err := sc.RedisStorage.GetTokenFromRedis()
	if err != nil {
		if err == redis.ErrNil {
			return true, nil
		}
		return false, err
	}
	return isJWTExpired(jwtToken, sc.secret)
}

// RenewToken will generate new Smooch JWT token.
func (sc *smoochClient) RenewToken() (string, error) {
	sc.mtx.Lock()
	defer sc.mtx.Unlock()

	jwtToken, err := GenerateJWT("app", sc.keyID, sc.secret)
	if err != nil {
		return "", err
	}

	err = sc.RedisStorage.SaveTokenToRedis(jwtToken, JWTExpiration)
	if err != nil {
		return "", err
	}

	return jwtToken, nil
}

func (sc *smoochClient) AddWebhookEventHandler(handler WebhookEventHandler) {
	sc.webhookEventHandlers = append(sc.webhookEventHandlers, handler)
}

func (sc *smoochClient) Send(userID string, message *Message) (*ResponsePayload, error) {
	if userID == "" {
		return nil, ErrUserIDEmpty
	}

	if message == nil {
		return nil, ErrMessageNil
	}

	if message.Role == "" {
		return nil, ErrMessageRoleEmpty
	}

	if message.Type == "" {
		return nil, ErrMessageTypeEmpty
	}

	url := sc.getURL(
		fmt.Sprintf("/v1.1/apps/%s/appusers/%s/messages", sc.appID, userID),
		nil,
	)

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(message)
	if err != nil {
		return nil, err
	}

	req, err := sc.createRequest(http.MethodPost, url, buf, nil)
	if err != nil {
		return nil, err
	}

	var responsePayload ResponsePayload
	err = sc.sendRequest(req, &responsePayload)
	if err != nil {
		return nil, err
	}

	return &responsePayload, nil
}

func (sc *smoochClient) VerifyRequest(r *http.Request) bool {
	givenSecret := r.Header.Get("X-Api-Key")
	return sc.verifySecret == givenSecret
}

func (sc *smoochClient) GetAppUser(userID string) (*AppUser, error) {
	url := sc.getURL(
		fmt.Sprintf("/v1.1/apps/%s/appusers/%s", sc.appID, userID),
		nil,
	)

	req, err := sc.createRequest(http.MethodGet, url, nil, nil)
	if err != nil {
		return nil, err
	}

	var response GetAppUserResponse
	err = sc.sendRequest(req, &response)
	if err != nil {
		return nil, err
	}

	return response.AppUser, nil
}

func (sc *smoochClient) UploadFileAttachment(filepath string, upload AttachmentUpload) (*Attachment, error) {
	r, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return sc.UploadAttachment(r, upload)

}
func (sc *smoochClient) UploadAttachment(r io.Reader, upload AttachmentUpload) (*Attachment, error) {

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
		fmt.Sprintf("/v1.1/apps/%s/attachments", sc.appID),
		queryParams,
	)

	formData := map[string]io.Reader{
		"source": r,
		"type":   strings.NewReader(upload.MIMEType),
	}

	req, err := sc.createMultipartRequest(url, formData)
	if err != nil {
		return nil, err
	}

	var response Attachment
	err = sc.sendRequest(req, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (sc *smoochClient) DeleteAttachment(attachment *Attachment) error {
	url := sc.getURL(
		fmt.Sprintf("/v1.1/apps/%s/attachments", sc.appID),
		nil,
	)

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(attachment)
	if err != nil {
		return err
	}

	req, err := sc.createRequest(http.MethodPost, url, buf, nil)
	if err != nil {
		return err
	}

	err = sc.sendRequest(req, nil)
	if err != nil {
		return err
	}

	return nil
}

func (sc *smoochClient) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost || !sc.VerifyRequest(r) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		sc.logger.Errorw("request body read failed", "err", err)
		return
	}

	var payload Payload
	err = json.Unmarshal(body, &payload)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		sc.logger.Errorw("could not decode response", "err", err)
		return
	}

	w.WriteHeader(http.StatusOK)

	sc.dispatch(&payload)
}

func (sc *smoochClient) dispatch(p *Payload) {
	for _, handler := range sc.webhookEventHandlers {
		handler(p)
	}
}

func (sc *smoochClient) getURL(endpoint string, values url.Values) string {
	rootURL := usRootURL
	if sc.region == RegionEU {
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

func (sc *smoochClient) createRequest(
	method string,
	url string,
	buf *bytes.Buffer,
	header http.Header) (*http.Request, error) {

	var req *http.Request
	var err error

	if header == nil {
		header = http.Header{}
	}

	if header.Get(contentTypeHeaderKey) == "" {
		header.Set(contentTypeHeaderKey, contentTypeJSON)
	}

	jwtToken, err := sc.RedisStorage.GetTokenFromRedis()
	if err != nil {
		return nil, err
	}
	header.Set(authorizationHeaderKey, fmt.Sprintf("Bearer %s", jwtToken))

	if buf == nil {
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, buf)
	}
	if err != nil {
		return nil, err
	}
	req.Header = header

	return req, nil
}

func (sc *smoochClient) createMultipartRequest(
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

func (sc *smoochClient) sendRequest(req *http.Request, v interface{}) error {
	response, err := sc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		if v != nil {
			err := json.NewDecoder(response.Body).Decode(&v)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return checkSmoochError(response)
}
