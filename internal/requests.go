package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type Request struct {
	Client  *http.Client
	Headers http.Header
}

type RespHeaders struct {
	Text http.Header
	Json map[string]interface{}
}

type Response struct {
	R       *http.Response
	Headers RespHeaders
	Cookies map[string]string
	Text    string
	Json    map[string]interface{}
}

func Requests() *Request {
	jar, _ := cookiejar.New(nil)

	return &Request{
		Client: &http.Client{
			Jar: jar,
		},
		Headers: http.Header{},
	}
}

func (r *Request) SetHeader(k, v string) {
	r.Headers.Set(k, v)
}

func (r *Request) DelHeader(k string) {
	r.Headers.Del(k)
}

func (r *Request) ClearHeaders() {
	r.Headers = http.Header{}
}

func (r *Request) cloneHeaders() http.Header {
	h := http.Header{}
	for k, v := range r.Headers {
		h[k] = append([]string(nil), v...)
	}
	return h
}

func (r *Request) headersToJson(headers http.Header) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range headers {
		if len(v) == 1 {
			out[k] = v[0]
		} else {
			out[k] = v
		}
	}
	return out
}

func (r *Request) Do(method, urlStr string, args ...interface{}) (Response, error) {

	var body io.Reader
	query := url.Values{}
	headers := r.cloneHeaders()

	for _, arg := range args {

		switch v := arg.(type) {

		case map[string]string:
			for k, val := range v {
				query.Set(k, val)
			}

		case map[string]interface{}:
			b, _ := json.Marshal(v)
			body = bytes.NewReader(b)
			headers.Set("Content-Type", "application/json")

		case url.Values:
			for k := range v {
				query.Set(k, v.Get(k))
			}

		case io.Reader:
			body = v

		case []byte:
			body = bytes.NewReader(v)

		case string:
			body = bytes.NewBufferString(v)

		default:
			// struct / *struct 自动 JSON
			b, err := json.Marshal(v)
			if err == nil {
				body = bytes.NewReader(b)
				headers.Set("Content-Type", "application/json")
			}
		}
	}

	if len(query) > 0 {
		u, err := url.Parse(urlStr)
		if err != nil {
			return Response{}, err
		}

		q := u.Query()
		for k := range query {
			q.Set(k, query.Get(k))
		}

		u.RawQuery = q.Encode()
		urlStr = u.String()
	}

	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return Response{}, err
	}

	req.Header = headers

	resp, err := r.Client.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}

	var jsonBody map[string]interface{}
	if err := json.Unmarshal(raw, &jsonBody); err != nil {
		jsonBody = nil
	}

	cookies := map[string]string{}
	for _, c := range resp.Cookies() {
		cookies[c.Name] = c.Value
	}

	return Response{
		R:       resp,
		Text:    string(raw),
		Json:    jsonBody,
		Cookies: cookies,
		Headers: RespHeaders{
			Text: resp.Header,
			Json: r.headersToJson(resp.Header),
		},
	}, nil
}

func (r *Request) Get(url string, args ...interface{}) (Response, error) {
	return r.Do(http.MethodGet, url, args...)
}

func (r *Request) Post(url string, args ...interface{}) (Response, error) {
	return r.Do(http.MethodPost, url, args...)
}

func (r *Request) Put(url string, args ...interface{}) (Response, error) {
	return r.Do(http.MethodPut, url, args...)
}

func (r *Request) Delete(url string, args ...interface{}) (Response, error) {
	return r.Do(http.MethodDelete, url, args...)
}
