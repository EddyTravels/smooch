package smooch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	ErrUserIDEmpty       = errors.New("user id is empty")
	ErrMessageNil        = errors.New("message is nil")
	ErrMessageRoleEmpty  = errors.New("message.Role is empty")
	ErrMessageTypeEmpty  = errors.New("message.Type is empty")
	ErrVerifySecretEmpty = errors.New("verify secret is empty")
)

const (
	RegionUS = "US"
	RegionEU = "EU"

	usRootURL = "https://api.smooch.io"
	euRootURL = "https://api.eu-1.smooch.io"
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
}

type WebhookEventHandler func(payload *Payload)

type Client interface {
	Handler() http.Handler
	AddWebhookEventHandler(handler WebhookEventHandler)
	Send(userID string, message *Message) (*ResponsePayload, error)
	VerifyRequest(r *http.Request) bool
	GetAppUser(userID string) (*AppUser, error)
}

type smoochClient struct {
	mux                  *http.ServeMux
	appID                string
	jwtToken             string
	verifySecret         string
	logger               Logger
	region               string
	webhookEventHandlers []WebhookEventHandler
	httpClient           *http.Client
}

func New(o Options) (*smoochClient, error) {
	if o.VerifySecret == "" {
		return nil, ErrVerifySecretEmpty
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

	jwtToken, err := GenerateJWT("app", o.KeyID, o.Secret)
	if err != nil {
		return nil, err
	}

	sc := &smoochClient{
		mux:          o.Mux,
		appID:        o.AppID,
		verifySecret: o.VerifySecret,
		logger:       o.Logger,
		region:       region,
		httpClient:   o.HttpClient,
		jwtToken:     jwtToken,
	}

	sc.mux.HandleFunc(o.WebhookURL, sc.handle)
	return sc, nil
}

func (sc *smoochClient) Handler() http.Handler {
	return sc.mux
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
	)

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(message)
	if err != nil {
		return nil, err
	}

	var responsePayload ResponsePayload
	err = sc.sendRequest(http.MethodPost, url, buf, &responsePayload)
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
	)

	var response GetAppUserResponse
	err := sc.sendRequest(http.MethodGet, url, nil, &response)
	if err != nil {
		return nil, err
	}

	return response.AppUser, nil
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

func (sc *smoochClient) getURL(endpoint string) string {
	rootURL := usRootURL
	if sc.region == RegionEU {
		rootURL = euRootURL
	}
	return fmt.Sprintf("%s%s", rootURL, endpoint)
}

func (sc *smoochClient) sendRequest(method string, url string, buf *bytes.Buffer, v interface{}) error {
	var req *http.Request
	var err error
	if buf == nil {
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, url, buf)
	}
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sc.jwtToken))

	response, err := sc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= 200 && response.StatusCode < 300 {
		err := json.NewDecoder(response.Body).Decode(&v)
		if err != nil {
			return err
		}

		return nil
	}
	return checkSmoochError(response)
}
