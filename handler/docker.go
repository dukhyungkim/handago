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
	"time"
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

	output, err := h.deploy("shared", templateParam.Name, templateParam)
	if err != nil {
		h.sendDeployResponse(templateParam.ToActionResponse(request.GetSpace(), err.Error()))
		return
	}
	h.sendDeployResponse(templateParam.ToActionResponse(request.GetSpace(), output))
}

func (h *Handler) HandleCompanyAction(company, host, base string, request *pbAct.ActionRequest) {
	templateParam := tm.NewCompanyDeployTemplate(company, host, base, request.GetReqDeploy())

	output, err := h.deploy(company, templateParam.Name, templateParam)
	if err != nil {
		h.sendDeployResponse(templateParam.ToActionResponse(request.GetSpace(), err.Error()))
		return
	}
	h.sendDeployResponse(templateParam.ToActionResponse(request.GetSpace(), output))
}

func (h *Handler) deploy(company, name string, templateParam interface{}) (string, error) {
	deployTemplate, err := h.loadTemplate(name)
	if err != nil {
		log.Println(err)
		return "", err
	}

	tpl, err := template.New(name).Parse(deployTemplate)
	if err != nil {
		log.Printf("failed to create template; %v\n", err)
		return "", err
	}

	var tplBuffer bytes.Buffer
	if err = tpl.Execute(&tplBuffer, templateParam); err != nil {
		log.Printf("failed to apply template; %v\n", err)
		return "", err
	}

	output, err := h.executeDockerCompose(company, name, tplBuffer)
	if err != nil {
		log.Printf("failed to execute docker-compose; %v\n", err)
		return "", err
	}

	return output, nil
}

func (h *Handler) executeDockerCompose(company, name string, tplBuffer bytes.Buffer) (string, error) {
	tplPath := fmt.Sprintf("/tmp/%s_%s.yaml", company, name)
	if err := os.WriteFile(tplPath, tplBuffer.Bytes(), 0600); err != nil {
		return "", err
	}

	const cmdDockerCompose = "docker-compose"

	if err := exec.Command(cmdDockerCompose, "-f", tplPath, "up", "-d").Run(); err != nil {
		return "", err
	}

	time.Sleep(5 * time.Second)

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

func (h *Handler) sendDeployResponse(response *pbAct.ActionResponse) {
	if err := h.streamClient.PublishResponse(response); err != nil {
		log.Println(err)
		return
	}
}
