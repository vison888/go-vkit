package natsx

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsClient struct {
	conn             *nats.Conn
	connectedHandler func(*nats.Conn)
	options          []nats.Option
	url              string
	user             string
	password         string
}

func NewNatsClient(url, user, password string, opts ...nats.Option) *NatsClient {
	nc := &NatsClient{
		url:      url,
		user:     user,
		password: password,
	}
	nc.options = opts
	err := nc.Conn(opts...)
	if err != nil {
		panic(err)
	}
	return nc
}

func (the *NatsClient) Conn(options ...nats.Option) (err error) {
	newOptions := make([]nats.Option, 0)
	newOptions = append(newOptions, nats.UserInfo(the.user, the.password))
	// 默认是重试60次，每次3秒。3*60都不成功，就不会再重连了，
	// 如果有一次成功，则会重试计数会归0
	newOptions = append(newOptions, nats.Timeout(time.Second*3))
	newOptions = append(newOptions, nats.MaxReconnects(60*3))
	for _, v := range options {
		newOptions = append(newOptions, v)
	}
	conn, err := nats.Connect(the.url, newOptions...)
	if err != nil {
		return err
	}

	the.conn = conn
	if the.connectedHandler != nil {
		the.connectedHandler(conn)
	}
	return
}

func (the *NatsClient) GetClient() *nats.Conn {
	return the.conn
}

func (the *NatsClient) SetConnectedHandler(cb func(c *nats.Conn)) {
	the.connectedHandler = cb
}

func (the *NatsClient) Subscribe(subj string, cb nats.MsgHandler) (*nats.Subscription, error) {
	return the.conn.Subscribe(subj, cb)
}

func (the *NatsClient) Publish(subj string, data []byte) error {
	err := the.conn.Publish(subj, data)
	if err != nil {
		reconnectErr := the.TryReconnect(err)
		if reconnectErr != nil {
			return fmt.Errorf("reconnect fail %s", reconnectErr.Error())
		}
		return the.conn.Publish(subj, data)
	}
	return err
}

func (the *NatsClient) PublishMsg(m *nats.Msg) error {
	err := the.conn.PublishMsg(m)
	if err != nil {
		reconnectErr := the.TryReconnect(err)
		if reconnectErr != nil {
			return fmt.Errorf("reconnect fail %s", reconnectErr.Error())
		}
		return the.conn.PublishMsg(m)
	}
	return nil
}

func (the *NatsClient) PublishRequest(subj, reply string, data []byte) error {
	err := the.conn.PublishRequest(subj, reply, data)
	if err != nil {
		reconnectErr := the.TryReconnect(err)
		if reconnectErr != nil {
			return fmt.Errorf("reconnect fail %s", reconnectErr.Error())
		}
		return the.conn.PublishRequest(subj, reply, data)
	}
	return nil
}

func (the *NatsClient) TryReconnect(in error) (err error) {
	switch in {
	case nats.ErrConnectionClosed: // 如果是掉线则重连
		return the.Conn(the.options...)
	}
	return nil
}
