package natsx

import (
	"time"

	"github.com/nats-io/nats.go"
)

type NatsClient struct {
	conn *nats.Conn
}

func NewNatsClient(url, user, password string) *NatsClient {
	conn, err := nats.Connect(url, nats.UserInfo(user, password), nats.Timeout(3*time.Second))
	if err != nil {
		panic(err)
	}

	nc := &NatsClient{
		conn: conn,
	}

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
