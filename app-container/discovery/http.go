package discovery

import (
	"io"
	"net/http"
	"net/url"
)

var httpGet = http.Get

func httpsOrHTTP(name string, insecure bool) (urlStr string, body io.ReadCloser, err error) {
	fetch := func(scheme string) (urlStr string, res *http.Response, err error) {
		u, err := url.Parse(scheme + "://" + name)
		if err != nil {
			return "", nil, err
		}
		u.RawQuery = "ac-discovery=1"
		urlStr = u.String()
		res, err = httpGet(urlStr)
		return
	}
	closeBody := func(res *http.Response) {
		if res != nil {
			res.Body.Close()
		}
	}
	urlStr, res, err := fetch("https")
	if err != nil || res.StatusCode != 200 {
		closeBody(res)
		if insecure {
			urlStr, res, err = fetch("http")
		}
	}
	if err != nil {
		closeBody(res)
		return "", nil, err
	}
	return urlStr, res.Body, nil
}
