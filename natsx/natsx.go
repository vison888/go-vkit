package natsx

//github.com/nats-io/nats-server/v2 v2.8.2 // indirect
import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/visonlv/go-vkit/config"
	"github.com/visonlv/go-vkit/logger"
)

type NatsClient struct {
	conn *nats.Conn
}

func NewDefault() *NatsClient {
	url := config.GetString("mq.nats.url")
	username := config.GetString("mq.nats.username")
	password := config.GetString("mq.nats.password")
	return NewClient(url, username, password)
}

func NewClient(url, user, password string) *NatsClient {
	conn, err := nats.Connect(url, nats.UserInfo(user, password), nats.Timeout(3*time.Second))
	if err != nil {
		panic(err)
	}

	nc := &NatsClient{
		conn: conn,
	}

	logger.Infof("[nats] url:%s user:%s password:%s init success", url, user, password)
	return nc
}

func (the *NatsClient) GetClient() *nats.Conn {
	return the.conn
}

func (the *NatsClient) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return the.conn.Subscribe(subj, cb)

}

func (the *NatsClient) Publish(subj string, data []byte) error {
	return the.conn.Publish(subj, data)
}

func (the *NatsClient) PublishMsg(m *nats.Msg) error {
	return the.conn.PublishMsg(m)
}

func (the *NatsClient) PublishRequest(subj, reply string, data []byte) error {
	return the.conn.PublishRequest(subj, reply, data)
}
