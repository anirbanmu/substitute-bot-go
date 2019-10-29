package reddit

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ugorji/go/codec"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	baseRedditUrl   = "https://www.reddit.com"
	apiBaseUrl      = "https://www.reddit.com/api"
	oauthApiBaseUrl = "https://oauth.reddit.com/api"
)

type Credentials struct {
	Username     string
	Password     string
	ClientId     string
	ClientSecret string
	UserAgent    string
}

type Api struct {
	creds        Credentials
	Client       *http.Client
	token        string
	grantTime    time.Time
	EncodeBuffer *bytes.Buffer
	Encoder      *codec.Encoder
	Decoder      *codec.Decoder
}

type basicAuth struct {
	user string
	pass string
}

func buildUrl(baseUrl string, path string, query *url.Values) (*url.URL, error) {
	base, err := url.Parse(baseUrl)
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

func postUrlEncodedForm(client *http.Client, fullUrl string, args *url.Values, headers *map[string]string, auth *basicAuth) ([]byte, error) {
	body := ""
	if args != nil {
		body = args.Encode()
	}

	req, err := http.NewRequest("POST", fullUrl, strings.NewReader(body))
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
		return nil, fmt.Errorf("%s returned %d", fullUrl, res.StatusCode)
	}

	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return resBody, nil
}

func (api *Api) postUrlEncodedForm(path string, query *url.Values) ([]byte, error) {
	if err := api.reAuth(); err != nil {
		return nil, err
	}

	apiUrl, err := buildUrl(oauthApiBaseUrl, path, nil)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{"User-Agent": api.creds.UserAgent, "Authorization": "bearer " + api.token}

	return postUrlEncodedForm(api.Client, apiUrl.String(), query, &headers, nil)
}

func (api *Api) getJson(path string, query *url.Values) ([]byte, error) {
	if err := api.reAuth(); err != nil {
		return nil, err
	}

	apiUrl, err := buildUrl(oauthApiBaseUrl, path, query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", apiUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("User-Agent", api.creds.UserAgent)
	req.Header.Add("Authorization", "bearer "+api.token)

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

func IsFullnameComment(fullname string) bool {
	return strings.HasPrefix(fullname, "t1_")
}

func (api *Api) GetComment(fullname string) (*Comment, error) {
	if !IsFullnameComment(fullname) {
		return nil, errors.New("full name given was not a comment")
	}

	res, err := api.getJson("/info", &url.Values{"id": {fullname}, "raw_json": {"1"}})
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

	api.Decoder.ResetBytes(res)
	err = api.Decoder.Decode(&parsed)
	api.Decoder.ResetBytes(nil)

	if err != nil {
		return nil, err
	}

	if parsed.Data.Children != nil && len(parsed.Data.Children) == 1 && parsed.Data.Children[0].Kind == "t1" {
		return &parsed.Data.Children[0].Cmt, nil
	}

	return nil, errors.New("Could not retrieve comment")
}

func (api *Api) PostComment(fullname string, bodyMarkdown string) (*Comment, error) {
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

	res, err := api.postUrlEncodedForm("/comment", &body)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	parsed := struct {
		Json struct {
			Errors [][]string `json:"errors"`
			Data   struct {
				Things []struct {
					Kind string  `json:"kind"`
					Cmt  Comment `json:"data"`
				} `json:"things"`
			} `json:"data"`
		} `json:"json"`
	}{}

	api.Decoder.ResetBytes(res)
	err = api.Decoder.Decode(&parsed)
	api.Decoder.ResetBytes(nil)

	if err != nil {
		return nil, err
	}

	if len(parsed.Json.Errors) > 0 {
		errorStrings := make([]string, 0, len(parsed.Json.Errors))
		for i := 0; i < len(parsed.Json.Errors); i++ {
			errorStrings = append(errorStrings, "["+strings.Join(parsed.Json.Errors[i], ", ")+"]")
		}
		return nil, errors.New("API errors: " + strings.Join(errorStrings, ", "))
	}

	if parsed.Json.Data.Things != nil && len(parsed.Json.Data.Things) == 1 && parsed.Json.Data.Things[0].Kind == "t1" {
		return &parsed.Json.Data.Things[0].Cmt, nil
	}

	return nil, errors.New("Could not post comment")
}

func (api *Api) reAuth() error {
	if time.Since(api.grantTime) < time.Minute*40 {
		return nil
	}

	log.Printf("reddit.Api - initiating re auth")
	token, err := auth(api.creds, api.Client)
	if err != nil {
		log.Printf("reddit.Api - failed to re auth: %s", err)
		return err
	}

	log.Printf("reddit.Api - successfully re authed")

	api.token = token
	api.grantTime = time.Now()
	return nil
}

func InitApiFromEnv(client *http.Client) (*Api, error) {
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

	if len(creds.ClientId) == 0 {
		return nil, errors.New("environment variable SUBSTITUTE_BOT_CLIENT_ID is required")
	}

	if len(creds.ClientSecret) == 0 {
		return nil, errors.New("environment variable SUBSTITUTE_BOT_CLIENT_SECRET is required")
	}

	if len(creds.UserAgent) == 0 {
		return nil, errors.New("environment variable SUBSTITUTE_BOT_USER_AGENT is required")
	}

	return InitApi(creds, client)
}

func InitApi(creds Credentials, client *http.Client) (*Api, error) {
	if client == nil {
		client = &http.Client{Timeout: time.Second * 10}
	}

	token, err := auth(creds, client)
	if err != nil {
		return nil, err
	}

	encodeBuffer := bytes.Buffer{}
	encoder := codec.NewEncoder(&encodeBuffer, &codec.JsonHandle{})

	return &Api{
		creds:        creds,
		Client:       client,
		token:        token,
		grantTime:    time.Now(),
		EncodeBuffer: &encodeBuffer,
		Encoder:      encoder,
		Decoder:      codec.NewDecoderBytes(nil, &codec.JsonHandle{}),
	}, nil
}

func buildAuthRequest(creds Credentials) (*http.Request, error) {
	reqBody := url.Values{
		"grant_type": {"password"},
		"username":   {creds.Username},
		"password":   {creds.Password},
	}

	apiUrl, err := buildUrl(apiBaseUrl, "/v1/access_token", nil)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", apiUrl.String(), strings.NewReader(reqBody.Encode()))
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(creds.ClientId, creds.ClientSecret)
	req.Header.Add("User-Agent", creds.UserAgent)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}

func auth(creds Credentials, client *http.Client) (string, error) {
	apiUrl, err := buildUrl(apiBaseUrl, "/v1/access_token", nil)
	if err != nil {
		return "", err
	}

	args := url.Values{
		"grant_type": {"password"},
		"username":   {creds.Username},
		"password":   {creds.Password},
	}

	headers := map[string]string{"User-Agent": creds.UserAgent}

	auth := basicAuth{creds.ClientId, creds.ClientSecret}

	res, err := postUrlEncodedForm(client, apiUrl.String(), &args, &headers, &auth)
	if err != nil {
		return "", err
	}

	parsed := struct {
		Token string `json:"access_token"`
	}{}

	decoder := codec.NewDecoderBytes(res, &codec.JsonHandle{})
	if err := decoder.Decode(&parsed); err != nil {
		return "", err
	}

	if len(parsed.Token) != 0 {
		return parsed.Token, nil
	}

	return "", errors.New("Reddit access_token API returned empty string")
}
