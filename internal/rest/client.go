package rest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eldelto/core/internal/web"
)

const ISO8601Format = "2006-01-02 15:04:05.000"

type ISO8601Time time.Time

func (t *ISO8601Time) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		*t = ISO8601Time(time.Time{})
		return nil
	}
	time, err := time.Parse(ISO8601Format, s)
	*t = ISO8601Time(time)
	return err
}

func (t *ISO8601Time) MarshalJSON() ([]byte, error) {
	time := time.Time(*t)
	if time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(time.Format(ISO8601Format)), nil
}

type Authenticator interface {
	Authenticate(r *http.Request) error
}

type BasicAuth struct {
	Username string
	Password string
}

func (b *BasicAuth) Authenticate(r *http.Request) error {
	credentials := base64.StdEncoding.EncodeToString([]byte(b.Username + ":" + b.Password))
	r.Header.Add("Authorization", "Basic "+credentials)
	return nil
}

type BearerAuth struct {
	Token string
}

func (b *BearerAuth) Authenticate(r *http.Request) error {
	r.Header.Add("Authorization", "Bearer "+b.Token)
	return nil
}

type HeaderAuth struct {
	Name  string
	Value string
}

func (h *HeaderAuth) Authenticate(r *http.Request) error {
	r.Header.Add(h.Name, h.Value)
	return nil
}

func jsonRequest(httpMethod string,
	url string,
	auth Authenticator,
	payload io.Reader,
	headers map[string]string,
	cookies []http.Cookie) (*http.Response, error) {
	request, err := http.NewRequest(httpMethod, url, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %q: %w", url, err)
	}

	if auth != nil {
		if err := auth.Authenticate(request); err != nil {
			return nil, fmt.Errorf("failed to authenticate request to %q: %w", url, err)
		}
	}

	request.Header.Add("Content-Type", "application/json")

	for key, value := range headers {
		request.Header.Add(key, value)
	}

	if host, ok := headers["Host"]; ok {
		request.Host = host
	}

	for i := range cookies {
		request.AddCookie(&cookies[i])
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request to %q: %w", url, err)
	}

	if response.StatusCode >= 300 {
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body for %q and status code %d: %w",
				url, response.StatusCode, err)
		}
		return nil, fmt.Errorf("request to %q returned unexpected response %d: %q",
			request.URL.String(), response.StatusCode, string(body))
	}

	return response, nil
}

func requestWithResponse(httpMethod, url string, auth Authenticator, requestData, responseData any, headers map[string]string) error {
	payload := bytes.Buffer{}
	if err := json.NewEncoder(&payload).Encode(requestData); err != nil {
		return fmt.Errorf("failed to encode request data for %q: %w", url, err)
	}

	response, err := jsonRequest(httpMethod, url, auth, &payload, headers, []http.Cookie{})
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if responseData == nil {
		return nil
	}

	if err := json.NewDecoder(response.Body).Decode(responseData); err != nil {
		return fmt.Errorf("failed to decode response from %q: %w", url, err)
	}

	return nil
}

// TODO: Refactor to request builder style or something like that.
func Get(url string, auth Authenticator, responseData any) error {
	return requestWithResponse(http.MethodGet, url, auth, nil, responseData, nil)
}

func GetWithHeader(url string, auth Authenticator, responseData any, headers map[string]string) error {
	return requestWithResponse(http.MethodGet, url, auth, nil, responseData, headers)
}

func Post(url string, auth Authenticator, requestData, responseData any, headers map[string]string) error {
	return requestWithResponse(http.MethodPost, url, auth, requestData, responseData, headers)
}

func Put(url string, auth Authenticator, requestData, responseData any) error {
	return requestWithResponse(http.MethodPut, url, auth, requestData, responseData, nil)
}

func Delete(url string, auth Authenticator) error {
	return requestWithResponse(http.MethodDelete, url, auth, nil, nil, nil)
}

type RequestBuilder struct {
	method       string
	url          *url.URL
	requestBody  any
	responseBody any
	headers      map[string]string
	cookies      []http.Cookie
	auth         Authenticator
}

func GET(url *url.URL) *RequestBuilder {
	return &RequestBuilder{
		method:  http.MethodGet,
		url:     url,
		headers: map[string]string{web.ContentTypeHeader: web.ContentTypeJSON},
	}
}

func (r *RequestBuilder) ResponseAs(responseBody any) *RequestBuilder {
	r.responseBody = &responseBody
	return r
}

func (r *RequestBuilder) AddHeader(key, value string) *RequestBuilder {
	r.headers[key] = value
	return r
}

func (r *RequestBuilder) Cookies(cookies []http.Cookie) *RequestBuilder {
	r.cookies = cookies
	return r
}

func (r *RequestBuilder) Auth(auth Authenticator) *RequestBuilder {
	r.auth = auth
	return r
}

func (r *RequestBuilder) Run() error {
	payload := bytes.Buffer{}
	if err := json.NewEncoder(&payload).Encode(r.requestBody); err != nil {
		return fmt.Errorf("failed to encode request data for %q: %w", r.url, err)
	}

	response, err := jsonRequest(r.method, r.url.String(), r.auth, &payload, r.headers, r.cookies)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if r.responseBody == nil {
		return nil
	}

	if err := json.NewDecoder(response.Body).Decode(r.responseBody); err != nil {
		return fmt.Errorf("failed to decode response from %q: %w", r.url, err)
	}

	return nil
}
