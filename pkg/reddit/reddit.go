package reddit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ugorji/go/codec"
)

const (
	baseRedditURL   = "https://www.reddit.com"
	apiBaseURL      = "https://www.reddit.com/api"
	oauthAPIBaseURL = "https://oauth.reddit.com/api"
)

// Credentials encapsulates the information needed for reddit API auth
type Credentials struct {
	Username     string
	Password     string
	ClientID     string
	ClientSecret string
	UserAgent    string
}

// API provides the abstraction to the reddit API
type API struct {
	creds     Credentials
	Client    *http.Client
	token     string
	grantTime time.Time
	mutex     sync.RWMutex
	Decoder   *codec.Decoder
}

type basicAuth struct {
	user string
	pass string
}

func buildURL(baseURL string, path string, query *url.Values) (*url.URL, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	p, err := url.Parse(base.Path + path)
	if err != nil {
		return nil, err
	}

	resolved := base.ResolveReference(p)

	if query != nil {
		resolved.RawQuery = query.Encode()
	}

	return resolved, nil
}

func postURLEncodedForm(client *http.Client, fullURL string, args *url.Values, headers *map[string]string, auth *basicAuth) ([]byte, error) {
	body := ""
	if args != nil {
		body = args.Encode()
	}

	req, err := http.NewRequest("POST", fullURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	if headers != nil {
		for k, v := range *headers {
			req.Header.Add(k, v)
		}
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if auth != nil {
		req.SetBasicAuth(auth.user, auth.pass)
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("%s returned %d", fullURL, res.StatusCode)
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return resBody, nil
}

func (api *API) postURLEncodedForm(path string, query *url.Values) ([]byte, error) {
	if err := api.reAuth(); err != nil {
		return nil, err
	}

	apiURL, err := buildURL(oauthAPIBaseURL, path, nil)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{"User-Agent": api.creds.UserAgent, "Authorization": "bearer " + api.authToken()}

	return postURLEncodedForm(api.Client, apiURL.String(), query, &headers, nil)
}

func (api *API) getJSON(path string, query *url.Values) ([]byte, error) {
	if err := api.reAuth(); err != nil {
		return nil, err
	}

	apiURL, err := buildURL(oauthAPIBaseURL, path, query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", apiURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", api.creds.UserAgent)
	req.Header.Add("Authorization", "bearer "+api.authToken())

	res, err := api.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Reddit %s API returned %d", path, res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// IsFullnameComment allows external checking of if a fullname represents a comment (beginning with t1_)
func IsFullnameComment(fullname string) bool {
	return strings.HasPrefix(fullname, "t1_")
}

// GetComment retrieves a comment by its fullname (t1_*) from the reddit API
func (api *API) GetComment(fullname string) (*Comment, error) {
	if !IsFullnameComment(fullname) {
		return nil, errors.New("full name given was not a comment")
	}

	res, err := api.getJSON("/info", &url.Values{"id": {fullname}, "raw_json": {"1"}})
	if err != nil {
		return nil, err
	}

	parsed := struct {
		Data struct {
			Children []struct {
				Kind string  `json:"kind"`
				Cmt  Comment `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}{}

	// api.Decoder.ResetBytes(res)
	// err = api.Decoder.Decode(&parsed)
	// api.Decoder.ResetBytes(nil)

	err = json.Unmarshal(res, &parsed)
	if err != nil {
		return nil, err
	}

	if parsed.Data.Children != nil && len(parsed.Data.Children) == 1 && parsed.Data.Children[0].Kind == "t1" {
		return &parsed.Data.Children[0].Cmt, nil
	}

	return nil, errors.New("Could not retrieve comment")
}

// PostComment posts a reply to the comment, referenced by fullname, with content of bodyMarkdown
func (api *API) PostComment(fullname string, bodyMarkdown string) (*Comment, error) {
	if len(fullname) == 0 {
		return nil, errors.New("fullname is blank")
	}

	if len(bodyMarkdown) == 0 {
		return nil, errors.New("body markdown text is blank")
	}

	body := url.Values{
		"raw_json": {"1"},
		"api_type": {"json"},
		"thing_id": {fullname},
		"text":     {bodyMarkdown},
	}

	res, err := api.postURLEncodedForm("/comment", &body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	parsed := struct {
		JSON struct {
			Errors [][]string `json:"errors"`
			Data   struct {
				Things []struct {
					Kind string  `json:"kind"`
					Cmt  Comment `json:"data"`
				} `json:"things"`
			} `json:"data"`
		} `json:"json"`
	}{}

	// api.Decoder.ResetBytes(res)
	// err = api.Decoder.Decode(&parsed)
	// api.Decoder.ResetBytes(nil)

	err = json.Unmarshal(res, &parsed)
	if err != nil {
		return nil, err
	}

	if len(parsed.JSON.Errors) > 0 {
		errorStrings := make([]string, 0, len(parsed.JSON.Errors))
		for i := 0; i < len(parsed.JSON.Errors); i++ {
			errorStrings = append(errorStrings, "["+strings.Join(parsed.JSON.Errors[i], ", ")+"]")
		}
		return nil, errors.New("API errors: " + strings.Join(errorStrings, ", "))
	}

	if parsed.JSON.Data.Things != nil && len(parsed.JSON.Data.Things) == 1 && parsed.JSON.Data.Things[0].Kind == "t1" {
		return &parsed.JSON.Data.Things[0].Cmt, nil
	}

	return nil, errors.New("Could not post comment")
}

func (api *API) authToken() string {
	api.mutex.RLock()
	defer api.mutex.RUnlock()

	return api.token
}

func (api *API) authValid() bool {
	return time.Since(api.grantTime) < 40*time.Minute
}

func (api *API) reAuth() error {
	api.mutex.RLock()
	valid := api.authValid()
	api.mutex.RUnlock()
	if valid {
		return nil
	}

	api.mutex.Lock()
	defer api.mutex.Unlock()

	// Check again to see if someone else acquired the write lock & reAuthed before us
	if api.authValid() {
		return nil
	}

	log.Printf("reddit.API - initiating re auth")
	token, err := auth(api.creds, api.Client)
	if err != nil {
		log.Printf("reddit.API - failed to re auth: %s", err)
		return err
	}

	log.Printf("reddit.API - successfully re authed")

	api.token = token
	api.grantTime = time.Now()
	return nil
}

// InitAPIFromEnv initializes (& auths) a reddit API client by reading credentials from the environment variables
func InitAPIFromEnv(client *http.Client) (*API, error) {
	creds := Credentials{
		os.Getenv("SUBSTITUTE_BOT_USERNAME"),
		os.Getenv("SUBSTITUTE_BOT_PASSWORD"),
		os.Getenv("SUBSTITUTE_BOT_CLIENT_ID"),
		os.Getenv("SUBSTITUTE_BOT_CLIENT_SECRET"),
		os.Getenv("SUBSTITUTE_BOT_USER_AGENT"),
	}

	if len(creds.Username) == 0 {
		return nil, errors.New("environment variable SUBSTITUTE_BOT_USERNAME is required")
	}

	if len(creds.Password) == 0 {
		return nil, errors.New("environment variable SUBSTITUTE_BOT_PASSWORD is required")
	}

	if len(creds.ClientID) == 0 {
		return nil, errors.New("environment variable SUBSTITUTE_BOT_CLIENT_ID is required")
	}

	if len(creds.ClientSecret) == 0 {
		return nil, errors.New("environment variable SUBSTITUTE_BOT_CLIENT_SECRET is required")
	}

	if len(creds.UserAgent) == 0 {
		return nil, errors.New("environment variable SUBSTITUTE_BOT_USER_AGENT is required")
	}

	return InitAPI(creds, client)
}

// InitAPI initializes (& auths) a reddit API client using the credentials provided
func InitAPI(creds Credentials, client *http.Client) (*API, error) {
	if client == nil {
		client = &http.Client{Timeout: time.Second * 10}
	}

	token, err := auth(creds, client)
	if err != nil {
		return nil, err
	}

	return &API{
		creds:     creds,
		Client:    client,
		token:     token,
		grantTime: time.Now(),
		Decoder:   codec.NewDecoderBytes(nil, &codec.JsonHandle{}),
	}, nil
}

func buildAuthRequest(creds Credentials) (*http.Request, error) {
	reqBody := url.Values{
		"grant_type": {"password"},
		"username":   {creds.Username},
		"password":   {creds.Password},
	}

	apiURL, err := buildURL(apiBaseURL, "/v1/access_token", nil)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiURL.String(), strings.NewReader(reqBody.Encode()))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(creds.ClientID, creds.ClientSecret)
	req.Header.Add("User-Agent", creds.UserAgent)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}

func auth(creds Credentials, client *http.Client) (string, error) {
	apiURL, err := buildURL(apiBaseURL, "/v1/access_token", nil)
	if err != nil {
		return "", err
	}

	args := url.Values{
		"grant_type": {"password"},
		"username":   {creds.Username},
		"password":   {creds.Password},
	}

	headers := map[string]string{"User-Agent": creds.UserAgent}

	auth := basicAuth{creds.ClientID, creds.ClientSecret}

	res, err := postURLEncodedForm(client, apiURL.String(), &args, &headers, &auth)
	if err != nil {
		return "", err
	}

	parsed := struct {
		Token string `json:"access_token"`
	}{}

	// decoder := codec.NewDecoderBytes(res, &codec.JsonHandle{})
	// if err := decoder.Decode(&parsed); err != nil {
	if err := json.Unmarshal(res, &parsed); err != nil {
		return "", err
	}

	if len(parsed.Token) != 0 {
		return parsed.Token, nil
	}

	return "", errors.New("Reddit access_token API returned empty string")
}
