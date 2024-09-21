package testutils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/eldelto/core/internal/web"
)

func AssertEquals(t testing.TB, expected, actual any, title string) {
	t.Helper()

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("%s should be\n'%v'\nbut was\n'%v'", title, expected, actual)
	}
}

func AssertNotEquals(t testing.TB, expected, actual any, title string) {
	t.Helper()

	if reflect.DeepEqual(expected, actual) {
		t.Errorf("%s should not be\n'%v'\nbut was\n'%v'", title, expected, actual)
	}
}

func AssertContains[T comparable](t testing.TB, expected T, testee []T, title string) {
	t.Helper()

	for _, actual := range testee {
		if reflect.DeepEqual(expected, actual) {
			return
		}
	}

	t.Errorf("%s did not contain a value '%v': %v", title, expected, testee)
}

func AssertContainsAll[T comparable](t testing.TB, expected []T, testee []T, title string) {
	t.Helper()

	// TODO: Think about if this is a good idea or not.
	// AssertEquals(t, len(expected), len(testee), "length of " + title)

	for i := range expected {
		AssertEquals(t, expected[i], testee[i], fmt.Sprintf("%s at index %d", title, i))
	}
}

func AssertStringContains(t testing.TB, expected, testee, title string) {
	t.Helper()

	if !strings.Contains(testee, expected) {
		t.Errorf("%s did not contain the substring '%v': %v", title, expected, testee)
	}
}

func AssertNoError(t testing.TB, err error, title string) {
	t.Helper()

	if err != nil {
		t.Errorf("%s should not return an error but returned '%v'", title, err)
	}
}

func AssertError(t testing.TB, err error, title string) {
	t.Helper()

	if err == nil {
		t.Errorf("%s should return an error but returned nil", title)
	}
}

func AssertNotNil(t testing.TB, actual any, title string) {
	t.Helper()

	if actual == nil {
		t.Errorf("%s should not be\n'nil'\nbut was\n'%v'", title, actual)
	}
}

type Response struct {
	response   *http.Response
	T          testing.TB
	StatusCode int
	mapBody    map[string]interface{}
}

func NewResponse(t testing.TB, response *http.Response) Response {
	return Response{
		response:   response,
		T:          t,
		StatusCode: response.StatusCode,
		mapBody:    map[string]interface{}{},
	}
}

func (r *Response) Body() map[string]interface{} {
	if len(r.mapBody) <= 0 {
		defer r.response.Body.Close()

		err := json.NewDecoder(r.response.Body).Decode(&r.mapBody)
		if err != nil && !errors.Is(err, io.EOF) {
			r.T.Fatalf("json.Decode error: %v", err)
		}
	}

	return r.mapBody
}

func (r *Response) Decode(value interface{}) error {
	defer r.response.Body.Close()

	return json.NewDecoder(r.response.Body).Decode(value)
}

type TestServer struct {
	*httptest.Server
	T      testing.TB
	Client *http.Client
}

func NewTestServer(t testing.TB, handler http.Handler) *TestServer {
	ts := httptest.NewServer(handler)
	return &TestServer{
		Server: ts,
		T:      t,
		Client: http.DefaultClient,
	}
}

func (ts *TestServer) GET(path string) Response {
	return ts.request("GET", path, "")
}

func (ts *TestServer) POST(path string, body string) Response {
	return ts.request("POST", path, body)
}

func (ts *TestServer) PUT(path string, body string) Response {
	return ts.request("PUT", path, body)
}

func (ts *TestServer) DELETE(path string) Response {
	return ts.request("DELETE", path, "")
}

func (ts *TestServer) request(verb, path string, body string) Response {
	url := ts.URL + path
	bodyData := bytes.NewBufferString(body)

	req, err := http.NewRequest(verb, url, bodyData)
	if err != nil {
		ts.T.Fatalf("http.NewRequest error: %v", err)
	}

	req.Header.Set(web.ContentTypeHeader, web.ContentTypeJSON)

	response, err := ts.Client.Do(req)
	if err != nil {
		ts.T.Fatalf("ts.Client.Do error: %v", err)
	}

	return NewResponse(ts.T, response)
}
