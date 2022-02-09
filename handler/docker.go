package handler

import (
	"bytes"
	"context"
	"fmt"
	pbAct "github.com/dukhyungkim/libharago/gen/go/proto/action"
	clientv3 "go.etcd.io/etcd/client/v3"
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

func (d *Docker) Close() {
	if err := d.etcdClient.Close(); err != nil {
		log.Printf("failed to close etcd client cleany; %v\n", err)
	}
}

func (d *Docker) HandleAction(request *pbAct.ActionRequest) {
	templateParam := tm.NewDeployTemplate(request.GetReqDeploy())

	d.deploy(templateParam.Name, request.GetSpace(), templateParam)
}

func (d *Docker) HandleCompanyAction(company string, host string, request *pbAct.ActionRequest) {
	templateParam := tm.NewCompanyDeployTemplate(company, host, request.GetReqDeploy())

	d.deploy(templateParam.Name, request.GetSpace(), templateParam)
}

func (d *Docker) deploy(name string, space string, templateParam interface{}) {
	deployTemplate, err := d.loadTemplate(name)
	if err != nil {
		log.Println(err)
		return
	}

	tpl, err := template.New(name).Parse(deployTemplate)
	if err != nil {
		log.Printf("failed to create template; %v\n", err)
		return
	}

	var tplBuffer bytes.Buffer
	if err = tpl.Execute(&tplBuffer, templateParam); err != nil {
		log.Println("failed to apply template ", err)
		return
	}

	output, err := d.executeDockerCompose(name, tplBuffer)
	if err != nil {
		log.Printf("failed to execute docker-compose; %v\n", err)
		return
	}

	d.sendResponse(space, output, err)
}

func (d *Docker) executeDockerCompose(name string, tplBuffer bytes.Buffer) (string, error) {
	tplPath := fmt.Sprintf("/tmp/%s.yaml", name)
	if err := ioutil.WriteFile(tplPath, tplBuffer.Bytes(), 0644); err != nil {
		return "", err
	}

	const cmdDockerCompose = "docker-compose"

	if err := exec.Command(cmdDockerCompose, "-f", tplPath, "up", "-d").Run(); err != nil {
		return "", err
	}

	output, err := exec.Command(cmdDockerCompose, "-f", tplPath, "ps").Output()
	if err != nil {
		return "", err
	}

	if err = os.Remove(tplPath); err != nil {
		return "", err
	}
	return string(output), err
}

func (d *Docker) loadTemplate(name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.DefaultTimeout)
	defer cancel()

	deployKey := fmt.Sprintf("/%s", name)
	deployTemplate, err := d.etcdClient.Get(ctx, deployKey)
	if err != nil {
		return "", err
	}

	if len(deployTemplate.Kvs) == 0 {
		return "", fmt.Errorf("failed to find value from key: %s", deployKey)
	}
	return string(deployTemplate.Kvs[0].Value), nil
}

func (d *Docker) sendResponse(space string, output string, err error) {
	response := &pbAct.ActionResponse{
		Text:  output,
		Space: space,
	}
	if err = d.streamClient.PublishResponse(response); err != nil {
		log.Println(err)
		return
	}
}
