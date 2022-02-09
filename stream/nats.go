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

type CompanyActionHandler func(company string, host string, request *pbAct.ActionRequest)

func (s *Client) ClamCompanyAction(company string, host string, handler CompanyActionHandler) error {
	if _, err := s.nc.Subscribe("harago.company.action", func(msg *nats.Msg) {
		var request pbAct.ActionRequest
		if err := proto.Unmarshal(msg.Data, &request); err != nil {
			log.Println(err)
			return
		}

		log.Println("Request:", request.String())
		handler(company, host, &request)
	}); err != nil {
		return err
	}

	specialSubject := fmt.Sprintf("harago.%s.action", company)
	if _, err := s.nc.QueueSubscribe(specialSubject, "handago", func(msg *nats.Msg) {
		var request pbAct.ActionRequest
		if err := proto.Unmarshal(msg.Data, &request); err != nil {
			log.Println(err)
			return
		}

		log.Println("Request:", request.String())
		handler(company, host, &request)
	}); err != nil {
		return err
	}

	return nil
}

type ActionHandler func(request *pbAct.ActionRequest)

func (s *Client) ClamSharedAction(handler ActionHandler) error {
	if _, err := s.nc.Subscribe("harago.shared.action", func(msg *nats.Msg) {
		var request pbAct.ActionRequest
		if err := proto.Unmarshal(msg.Data, &request); err != nil {
			log.Println(err)
			return
		}

		log.Println("Request:", request.String())
		handler(&request)
	}); err != nil {
		return err
	}
	return nil
}
