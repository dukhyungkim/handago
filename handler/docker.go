package handler

import (
	"bytes"
	"context"
	"fmt"
	pbAct "github.com/dukhyungkim/libharago/gen/go/proto/action"
	clientv3 "go.etcd.io/etcd/client/v3"
	"handago/common"
	"handago/config"
	tm "handago/handler/model"
	"handago/stream"
	"html/template"
	"log"
	"os"
	"os/exec"
)

type Handler struct {
	etcdClient   *clientv3.Client
	streamClient *stream.Client
}

func NewHandler(cfg *config.Etcd, streamClient *stream.Client) (*Handler, error) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: common.DefaultTimeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, common.ErrConnEtcd(err)
	}

	return &Handler{etcdClient: etcdClient, streamClient: streamClient}, nil
}

func (h *Handler) Close() {
	if err := h.etcdClient.Close(); err != nil {
		log.Printf("failed to close etcd client cleany; %v\n", err)
	}
}

func (h *Handler) HandleSharedAction(host, base string, request *pbAct.ActionRequest) {
	templateParam := tm.NewSharedDeployTemplate(host, base, request.GetReqDeploy())

	h.deploy(templateParam.Name, request.GetSpace(), templateParam)
}

func (h *Handler) HandleCompanyAction(company, host, base string, request *pbAct.ActionRequest) {
	templateParam := tm.NewCompanyDeployTemplate(company, host, base, request.GetReqDeploy())

	h.deploy(templateParam.Name, request.GetSpace(), templateParam)
}

func (h *Handler) deploy(name, space string, templateParam interface{}) {
	deployTemplate, err := h.loadTemplate(name)
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

	output, err := h.executeDockerCompose(name, tplBuffer)
	if err != nil {
		log.Printf("failed to execute docker-compose; %v\n", err)
		return
	}

	h.sendResponse(space, output)
}

func (h *Handler) executeDockerCompose(name string, tplBuffer bytes.Buffer) (string, error) {
	tplPath := fmt.Sprintf("/tmp/%s.yaml", name)
	if err := os.WriteFile(tplPath, tplBuffer.Bytes(), 0600); err != nil {
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

	err = os.Remove(tplPath)
	if err != nil {
		return "", err
	}
	return string(output), err
}

func (h *Handler) loadTemplate(name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.DefaultTimeout)
	defer cancel()

	deployKey := fmt.Sprintf("/%s", name)
	deployTemplate, err := h.etcdClient.Get(ctx, deployKey)
	if err != nil {
		return "", err
	}

	if len(deployTemplate.Kvs) == 0 {
		return "", fmt.Errorf("failed to find value from key: %s", deployKey)
	}
	return string(deployTemplate.Kvs[0].Value), nil
}

func (h *Handler) sendResponse(space, output string) {
	response := &pbAct.ActionResponse{
		Text:  output,
		Space: space,
	}
	if err := h.streamClient.PublishResponse(response); err != nil {
		log.Println(err)
		return
	}
}
