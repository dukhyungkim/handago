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

const (
	responseSubject = "handago.response"

	SharedSubject          = "harago.shared.action"
	CompanySubject         = "harago.company.action"
	InternalSubject        = "harago.internal.action"
	ExternalSubject        = "harago.external.action"
	SpecificCompanySubject = "harago.%s.action"
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

	err := s.nc.Publish(responseSubject, b)
	if err != nil {
		return err
	}
	return nil
}

type UpDownActionHandler func(request *pbAct.ActionRequest)

func (s *Client) ClamCompanyAction(company string, handler UpDownActionHandler) error {
	if _, err := s.nc.Subscribe(CompanySubject, runAction(handler)); err != nil {
		return err
	}

	specificCompanySubject := fmt.Sprintf(SpecificCompanySubject, company)
	if _, err := s.nc.QueueSubscribe(specificCompanySubject, "handago", runAction(handler)); err != nil {
		return err
	}
	return nil
}

func (s *Client) ClamSharedAction(handler UpDownActionHandler) error {
	if _, err := s.nc.Subscribe(SharedSubject, runAction(handler)); err != nil {
		return err
	}
	return nil
}

func (s *Client) ClamInternalAction(handler UpDownActionHandler) error {
	if _, err := s.nc.Subscribe(InternalSubject, runAction(handler)); err != nil {
		return err
	}
	return nil
}

func (s *Client) ClamExternalAction(handler UpDownActionHandler) error {
	if _, err := s.nc.Subscribe(ExternalSubject, runAction(handler)); err != nil {
		return err
	}
	return nil
}

func runAction(handler UpDownActionHandler) func(msg *nats.Msg) {
	return func(msg *nats.Msg) {
		var request pbAct.ActionRequest
		if err := proto.Unmarshal(msg.Data, &request); err != nil {
			log.Println(err)
			return
		}

		log.Println("Request:", request.String())
		handler(&request)
	}
}
