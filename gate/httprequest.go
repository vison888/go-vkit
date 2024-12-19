package gate

import (
	"net/http"

	"github.com/vison888/go-vkit/grpcx"
)

type HttpRequest struct {
	uri         string
	r           *http.Request
	service     string
	method      string
	contentType string
	body        []byte
	hasRead     bool
	fileMap     map[string]*grpcx.FileInfo
}

func (r *HttpRequest) Request() *http.Request {
	return r.r
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

func (r *HttpRequest) Read() ([]byte, map[string]*grpcx.FileInfo, error) {
	if r.hasRead {
		return r.body, r.fileMap, nil
	}
	b, fs, err := requestPayload(r.r)
	if err == nil {
		r.fileMap = fs
		r.hasRead = true
		r.body = b
	}
	return b, fs, err
}

func (r *HttpRequest) SetBody(b []byte) {
	r.hasRead = true
	r.body = b
}
