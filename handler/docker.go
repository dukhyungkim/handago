package handler

import (
	"bytes"
	"context"
	"fmt"
	pbAct "github.com/dukhyungkim/libharago/gen/go/proto/action"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
	"handago/common"
	"handago/config"
	tm "handago/handler/template_model"
	"handago/stream"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

type Docker struct {
	etcdClient   *clientv3.Client
	streamClient *stream.Client
}

func NewDockerHandler(cfg *config.Etcd, streamClient *stream.Client) (*Docker, error) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: common.DefaultTimeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, common.ErrConnEtcd(err)
	}

	return &Docker{etcdClient: etcdClient, streamClient: streamClient}, nil
}

func (d *Docker) HandleAction(_ string, data []byte) {
	var pbAction pbAct.ActionRequest
	if err := proto.Unmarshal(data, &pbAction); err != nil {
		log.Println(err)
		return
	}
	log.Println("Data:", pbAction.String())

	deployTeemplateParam := tm.NewDeployTemplate(pbAction.GetReqDeploy())

	ctx, cancel := context.WithTimeout(context.Background(), common.DefaultTimeout)
	defer cancel()

	deployKey := fmt.Sprintf("/%s", deployTeemplateParam.Name)
	deployTemplate, err := d.etcdClient.Get(ctx, deployKey)
	if err != nil {
		log.Println(fmt.Errorf("failed to get kv; %w", err))
		return
	}

	if len(deployTemplate.Kvs) == 0 {
		log.Println(fmt.Errorf("failed to find value from key: %s", deployKey))
	}

	tpl, err := template.New(deployTeemplateParam.Name).Parse(string(deployTemplate.Kvs[0].Value))
	if err != nil {
		log.Println(err)
		return
	}

	var tplBuffer bytes.Buffer
	if err = tpl.Execute(&tplBuffer, deployTeemplateParam); err != nil {
		log.Println(err)
		return
	}

	tplPath := fmt.Sprintf("/tmp/%s.yaml", deployTeemplateParam.Name)
	if err = ioutil.WriteFile(tplPath, tplBuffer.Bytes(), 0644); err != nil {
		log.Println(err)
		return
	}

	const cmdDockerCompose = "docker-compose"

	if err = exec.Command(cmdDockerCompose, "-f", tplPath, "up", "-d").Run(); err != nil {
		log.Println(err)
		return
	}

	output, err := exec.Command(cmdDockerCompose, "-f", tplPath, "ps").Output()
	if err != nil {
		log.Println(err)
		return
	}

	if err = os.Remove(tplPath); err != nil {
		log.Println(err)
		return
	}

	pbResponse := &pbAct.ActionResponse{
		Text:  string(output),
		Space: pbAction.GetSpace(),
	}
	if err = d.streamClient.PublishResponse(pbResponse); err != nil {
		log.Println(err)
		return
	}
}

func (d *Docker) Close() {
	if err := d.etcdClient.Close(); err != nil {
		log.Printf("failed to close etcd client cleany; %v\n", err)
	}
}
