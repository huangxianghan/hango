package restful

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
)

var defClient *http.Client

type HttpMethod = string

const (
	MethodGet    HttpMethod = "GET"
	MethodPost   HttpMethod = "POST"
	MethodPut    HttpMethod = "PUT"
	MethodPatch  HttpMethod = "PATCH" // RFC 5789
	MethodDelete HttpMethod = "DELETE"
)

func init() {
	defClient = &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Millisecond, //限制建立TCP连接的时间
				KeepAlive: 10 * time.Millisecond, //指定 TCP keep-alive 探测发送到对等方的频率。
			}).DialContext,
			ForceAttemptHTTP2:     true,                  //是否启用HTTP/2
			IdleConnTimeout:       90 * time.Millisecond, //连接多少时间没有使用则被关闭
			TLSHandshakeTimeout:   10 * time.Second,      //tls协商的超时时间
			ExpectContinueTimeout: 1 * time.Second,       //等待收到一个go-ahead响应报文所用的时间
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   16,
			MaxConnsPerHost:       16,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func Get[T any](url string, header map[string]string) (*T, error) {
	return Request[T](MethodGet, url, header)
}

func Post[T any](url string, header map[string]string, jsonBody ...any) (*T, error) {
	return Request[T](MethodPost, url, header, jsonBody...)
}

func Put[T any](url string, header map[string]string, jsonBody ...any) (*T, error) {
	return Request[T](MethodPut, url, header, jsonBody...)
}

func Patch[T any](url string, header map[string]string, jsonBody ...any) (*T, error) {
	return Request[T](MethodPatch, url, header, jsonBody...)
}

func Delete[T any](url string, header map[string]string, jsonBody ...any) (*T, error) {
	return Request[T](MethodDelete, url, header, jsonBody...)
}

func Request[T any](method HttpMethod, url string, header map[string]string, jsonBody ...any) (*T, error) {

	var (
		err        error
		request    *http.Request
		bodyReader io.Reader
	)

	if method != MethodGet && method != "" {
		if len(jsonBody) > 0 {
			body, err := sonic.Marshal(jsonBody[0])
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewReader(body)
		}

	}

	request, err = http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if method != MethodGet && method != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	if header != nil {
		for k, v := range header {
			request.Header.Set(k, v)
		}
	}

	resp, err := defClient.Do(request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var respBody []byte
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("code: %d ,status:%s,body:%s", resp.StatusCode, resp.Status, respBody)
	}

	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := new(T)
	err = sonic.Unmarshal(respBody, result)

	return result, err
}
