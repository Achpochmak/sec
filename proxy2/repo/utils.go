package repo

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

func createHTTPRequest(data *RequestData) (*http.Request, error) {
	req, err := http.NewRequest(
		data.Method,
		data.Scheme+"://"+data.Host+data.Path,
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Host = data.Host
	req.Header = convertFromBSON(data.Headers)
	addCookies(req, data.Cookies)
	req.URL.RawQuery = encodeQueryParams(convertFromBSON(data.GetParams))
	req.Body = createRequestBody(req, data)

	return req, nil
}

func addCookies(req *http.Request, cookies map[string]string) {
	for name, value := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
		})
	}
}

func encodeQueryParams(values map[string][]string) string {
	return url.Values(values).Encode()
}

func createRequestBody(req *http.Request, data *RequestData) io.ReadCloser {
	if req.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		params := convertFromBSON(data.PostParams)
		return io.NopCloser(strings.NewReader(url.Values(params).Encode()))
	}
	return io.NopCloser(strings.NewReader(data.Body))
}

func parseHTTPHeaders(headers http.Header) bson.M {
	result := convertToBSON(headers)
	delete(result, "Cookie")
	return result
}

func parseURLQuery(input *url.URL) bson.M {
	input.RawQuery = strings.ReplaceAll(input.RawQuery, ";", "&")
	return convertToBSON(input.Query())
}

func parseHTTPCookies(cookies []*http.Cookie) map[string]string {
	result := make(map[string]string, len(cookies))
	for _, cookie := range cookies {
		result[cookie.Name] = cookie.Value
	}
	return result
}

func parsePostParameters(req *http.Request) (bson.M, error) {
	if req.Body == nil {
		return nil, nil
	}

	if err := req.ParseForm(); err != nil {
		return nil, err
	}

	return convertToBSON(req.PostForm), nil
}
