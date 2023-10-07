package gate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/grpcx"
	"github.com/visonlv/go-vkit/logger"
	"github.com/visonlv/go-vkit/metadata"
	meta "github.com/visonlv/go-vkit/metadata"
)

func ErrorResponse(w http.ResponseWriter, r *http.Request, _err error) {
	var netErr *neterrors.NetError
	switch verr := _err.(type) {
	case *neterrors.NetError:
		netErr = verr
	default:
		netErr = &neterrors.NetError{
			Msg:    "系统错误",
			Code:   -1,
			Status: 400,
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(int(netErr.Status))

	paramJson, err := json.Marshal(*netErr)
	if err != nil {
		logger.Errorf("[gate] encode json fail url:%s", r.RequestURI)
		return
	}

	paramStr := string(paramJson)
	w.Header().Set("Content-Length", strconv.Itoa(len(paramJson)))
	logger.Errorf("[gate] with error ret:%s url:%s", paramStr, r.RequestURI)
	fmt.Fprintln(w, paramStr)
}

func requestPayload(r *http.Request) (bytes []byte, fileMap map[string]*grpcx.FileInfo, err error) {
	closeBody := func(body io.ReadCloser) {
		if e := body.Close(); e != nil {
			err = errors.New("[httphandler] body close failed")
			return
		}
	}

	ct := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(ct, "application/json"):
		defer closeBody(r.Body)
		bytes, err = io.ReadAll(r.Body)
		return
	case strings.Contains(ct, "application/x-www-form-urlencoded"):
		r.ParseForm()
		vals := make(map[string]string)
		for k, v := range r.Form {
			vals[k] = strings.Join(v, ",")
		}
		b, err := json.Marshal(vals)
		return b, nil, err
	case strings.Contains(ct, "multipart/form-data"):
		if err := r.ParseMultipartForm(int64(10 << 20)); err != nil {
			return nil, nil, err
		}
		vals := make(map[string]any)
		for k, v := range r.MultipartForm.Value {
			vals[k] = strings.Join(v, ",")
		}

		var files map[string]*grpcx.FileInfo
		if len(r.MultipartForm.File) > 0 {
			files = make(map[string]*grpcx.FileInfo)
		}
		for k := range r.MultipartForm.File {
			f, h, err := r.FormFile(k)
			if err != nil {
				return nil, nil, err
			}
			b1, err := io.ReadAll(f)
			if err != nil {
				return nil, nil, err
			}
			files[k] = &grpcx.FileInfo{
				Filename: h.Filename,
				Size:     h.Size,
				Content:  b1,
			}
		}
		b, err := json.Marshal(vals)
		return b, files, err
	default:
		err = fmt.Errorf("[httphandler] not support contentType:%s", ct)
		return
	}
}

func requestToContext(ctx context.Context, md meta.Metadata, r *http.Request) context.Context {
	for k, v := range r.Header {
		if k == "Connection" {
			continue
		}
		md[strings.ToLower(k)] = strings.Join(v, ",")
	}
	return metadata.NewContext(ctx, md)
}
