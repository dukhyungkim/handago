package stream

import (
	"fmt"
	"handago/config"
	"log"
	"strings"
	"time"

	pbAct "github.com/dukhyungkim/libharago/gen/go/proto/action"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
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
	b, _ := proto.Marshal(response)

	const subject = "handago.response"
	err := s.nc.Publish(subject, b)
	if err != nil {
		return err
	}
	return nil
}

type UpDownActionHandler func(request *pbAct.ActionRequest)

func (s *Client) ClamCompanyAction(company string, handler UpDownActionHandler) error {
	const commonCompanySubject = "harago.company.action"
	if _, err := s.nc.Subscribe(commonCompanySubject, func(msg *nats.Msg) {
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

	specificCompanySubject := fmt.Sprintf("harago.%s.action", company)
	if _, err := s.nc.QueueSubscribe(specificCompanySubject, "handago", func(msg *nats.Msg) {
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

func (s *Client) ClamSharedAction(handler UpDownActionHandler) error {
	const sharedActionSubject = "harago.shared.action"
	if _, err := s.nc.Subscribe(sharedActionSubject, func(msg *nats.Msg) {
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
