package stream

import (
	"fmt"
	pbAct "github.com/dukhyungkim/libharago/gen/go/proto/action"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
	"handago/config"
	"log"
	"strings"
	"time"
)

type Client struct {
	nc      *nats.Conn
	timeout time.Duration
}

func NewStreamClient(cfg *config.Nats) (*Client, error) {
	nc, err := nats.Connect(strings.Join(cfg.Servers, ","),
		nats.UserInfo(cfg.Username, cfg.Password))
	if err != nil {
		return nil, err
	}

	return &Client{nc: nc, timeout: cfg.Timeout}, nil
}

func (s *Client) Close() {
	s.nc.Close()
}

func (s *Client) PublishResponse(response *pbAct.ActionResponse) error {
	b, err := proto.Marshal(response)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("handago.response")
	if err := s.nc.Publish(subject, b); err != nil {
		return err
	}
	return nil
}

type ActionHandler func(request *pbAct.ActionRequest)

func (s *Client) ClamCompanyAction(handler ActionHandler) error {
	if _, err := s.nc.QueueSubscribe("harago.company.action", "handago", func(msg *nats.Msg) {
		var request pbAct.ActionRequest
		if err := proto.Unmarshal(msg.Data, &request); err != nil {
			log.Println(err)
			return
		}
		handler(&request)
	}); err != nil {
		return err
	}
	return nil
}

func (s *Client) ClamSharedAction(handler ActionHandler) error {
	if _, err := s.nc.Subscribe("harago.shared.action", func(msg *nats.Msg) {
		var request pbAct.ActionRequest
		if err := proto.Unmarshal(msg.Data, &request); err != nil {
			log.Println(err)
			return
		}
		handler(&request)
	}); err != nil {
		return err
	}
	return nil
}
