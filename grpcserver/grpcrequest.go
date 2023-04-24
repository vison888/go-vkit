package grpcserver

type GrpcRequest struct {
	service     string
	method      string
	contentType string
	stream      bool
	payload     interface{}
}

func (r *GrpcRequest) Service() string {
	return r.service
}

func (r *GrpcRequest) Method() string {
	return r.method
}

func (r *GrpcRequest) ContentType() string {
	return r.contentType
}

func (r *GrpcRequest) Stream() bool {
	return r.stream
}

func (r *GrpcRequest) Payload() interface{} {
	return r.payload
}
