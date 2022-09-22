package miniox

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/visonlv/go-vkit/logger"
)

type MinioClient struct {
	Client       *minio.Client
	DoMain       string // 返回的url，因为部署方式的原因，集群内外要用不同的minio地址
	EndPoint     string
	AccessKey    string
	AccessSecret string
	BucketName   string
}

func NewClient(doMain, endPoint, accessKey, accessSecret, bucketName string) (*MinioClient, error) {
	c := &MinioClient{
		DoMain:       doMain,
		EndPoint:     endPoint,
		AccessKey:    accessKey,
		AccessSecret: accessSecret,
		BucketName:   bucketName,
	}

	// 1. 获取客户端
	client, err := minio.New(c.EndPoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKey, c.AccessSecret, ""),
		Secure: false,
	})
	if err != nil {
		logger.Errorf("[miniox] NewClient fail:%s doMain:%s endPoint:%s accessKey:%s accessSecret:%s bucketName:%s", err.Error(), doMain, endPoint, accessKey, accessSecret, bucketName)
		return nil, err
	}

	c.Client = client

	ctx := context.Background()

	// 2. 检查桶是否存在
	exists, errBucketExists := c.Client.BucketExists(ctx, c.BucketName)
	if errBucketExists == nil && exists {
		return c, nil
	}

	// 创建桶
	err = c.Client.MakeBucket(ctx, c.BucketName, minio.MakeBucketOptions{})
	if err != nil {
		logger.Errorf("[miniox] NewClient create bucket fail:%s doMain:%s endPoint:%s accessKey:%s accessSecret:%s bucketName:%s", err.Error(), doMain, endPoint, accessKey, accessSecret, bucketName)
		return nil, err
	}

	logger.Errorf("[miniox] NewClient success doMain:%s endPoint:%s accessKey:%s accessSecret:%s bucketName:%s", err.Error(), doMain, endPoint, accessKey, accessSecret, bucketName)
	return c, nil
}

// file: source file path . object: an object's whole name which use to be the path in minio(eg. bucket/object(xxx/xxx/xx.xx))
func (the *MinioClient) UploadLocalFile(file string, object string) (url string, err error) {

	ctx := context.Background()

	// 1. 剥离后缀
	file_suffix := path.Ext(file)
	if file_suffix == "" {
		file_suffix = ".noSuffix"
	}

	// 2. 组装content-type
	content_type := "application/" + string([]byte(file_suffix)[1:])

	// 3. 存储
	_, err = the.Client.FPutObject(ctx, the.BucketName, object, file, minio.PutObjectOptions{ContentType: content_type})
	if err != nil {
		return
	}

	url = the.DoMain + "/" + the.BucketName + "/" + object

	return
}

// file: source file path . object: an object's whole name which use to be the path in minio(eg. bucket/object(xxx/xxx/xx.xx))
func (the *MinioClient) UploadStreamFile(stream []byte, object string) (url string, err error) {
	ctx := context.Background()

	// 2. 组装content-type
	content_type := http.DetectContentType(stream)

	// 3. 构造io.reader
	reader := bytes.NewBuffer(stream)

	// 4. 存储
	_, err = the.Client.PutObject(ctx, the.BucketName, object, reader, int64(len(stream)), minio.PutObjectOptions{ContentType: content_type})
	if err != nil {
		return
	}

	url = the.DoMain + "/" + the.BucketName + "/" + object

	return
}

func (m *MinioClient) DeleteFile(object string) (err error) {
	ctx := context.Background()

	err = m.Client.RemoveObject(ctx, m.BucketName, object, minio.RemoveObjectOptions{})

	return
}

func (m *MinioClient) DeleteFileByUrl(file_url string) (err error) {

	u, err := url.Parse(file_url)
	if err != nil {
		return err
	}

	s := strings.Split(u.EscapedPath(), "/")
	// 第一个是空格
	if len(s) < 3 {
		return errors.New("url" + file_url + " is invalid")
	}

	bucket := s[1]
	object := strings.Join(s[2:], "/")

	err = m.Client.RemoveObject(context.Background(), bucket, object, minio.RemoveObjectOptions{})
	if err != nil {
		return
	}

	return
}
