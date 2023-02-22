package gate

import (
	"net/http"
)

type HttpRequest struct {
	uri         string
	r           *http.Request
	service     string
	method      string
	contentType string
	body        []byte
	hasRead     bool
}

func (r *HttpRequest) ContentType() string {
	return r.contentType
}

func (r *HttpRequest) Service() string {
	return r.service
}

func (r *HttpRequest) Method() string {
	return r.method
}

func (r *HttpRequest) Endpoint() string {
	return r.method
}

func (r *HttpRequest) Uri() string {
	return r.uri
}

func (r *HttpRequest) Read() ([]byte, error) {
	if r.hasRead {
		return r.body, nil
	}
	b, err := requestPayload(r.r)
	if err == nil {
		r.hasRead = true
		r.body = b
	}
	return b, err
}
