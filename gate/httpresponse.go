package gate

import (
	"net/http"

	"github.com/visonlv/go-vkit/logger"
)

type HttpResponse struct {
	w        http.ResponseWriter
	header   map[string]string
	hasWrite bool
	content  []byte
}

func (r *HttpResponse) ResponseWriter() http.ResponseWriter {
	return r.w
}

func (r *HttpResponse) WriteHeader(hdr map[string]string) {
	for k, v := range hdr {
		r.header[k] = v
	}
}

func (r *HttpResponse) Write(b []byte) error {
	_, err := r.w.Write(b)
	if err != nil {
		logger.Infof("write fail content:%s err:%s", string(b), err.Error())
	}
	r.hasWrite = true
	return err
}

func (r *HttpResponse) Content() []byte {
	return r.content
}
