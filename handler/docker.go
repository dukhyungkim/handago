package handler

import (
	"bytes"
	"context"
	"fmt"
	"handago/common"
	"handago/config"
	"handago/handler/model"
	"handago/stream"
	"html/template"
	"log"
	"os"
	"os/exec"
	"time"

	pbAct "github.com/dukhyungkim/libharago/gen/go/proto/action"
	clientv3 "go.etcd.io/etcd/client/v3"
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
	company := "shared"
	actionType := request.GetType()
	switch actionType {
	case pbAct.ActionType_UP, pbAct.ActionType_DOWN:
		templateParam := model.NewDeployTemplateParam(host, base, request.GetReqDeploy())

		templateBuffer, err := h.prepareTemplate(templateParam)
		if err != nil {
			h.sendDeployResponse(templateParam.ToActionResponse(request.GetSpace(), err.Error(), actionType))
			return
		}

		output, err := h.executeDockerCompose(templateParam, templateBuffer, actionType)
		if err != nil {
			h.sendDeployResponse(templateParam.ToActionResponse(request.GetSpace(), err.Error(), actionType))
			return
		}

		h.sendDeployResponse(templateParam.ToActionResponse(request.GetSpace(), output, actionType))
	}
}

func (h *Handler) HandleCompanyAction(company, host, base string, request *pbAct.ActionRequest) {
	templateParam := model.NewDeployTemplateParam(host, base, request.GetReqDeploy())
	templateParam.SetCompany(company)

	output, err := h.prepareTemplate(templateParam)
	if err != nil {
		h.sendDeployResponse(templateParam.ToActionResponse(request.GetSpace(), err.Error()))
		return
	}
	h.sendDeployResponse(templateParam.ToActionResponse(request.GetSpace(), output))
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

func (h *Handler) prepareTemplate(templateParam *model.DeployTemplateParam) (*bytes.Buffer, error) {
	deployTemplate, err := h.loadTemplate(templateParam.Name)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	tpl, err := template.New(templateParam.Name).Parse(deployTemplate)
	if err != nil {
		log.Printf("failed to create template; %v\n", err)
		return nil, err
	}

	var tplBuffer bytes.Buffer
	err = tpl.Execute(&tplBuffer, templateParam)
	if err != nil {
		log.Printf("failed to apply template; %v\n", err)
		return nil, err
	}

	return &tplBuffer, nil
}

func (h *Handler) executeDockerCompose(templateParam *model.DeployTemplateParam, tplBuffer *bytes.Buffer, actionType pbAct.ActionType) (string, error) {
	var err error
	tplPath := fmt.Sprintf("/tmp/%s_%s.yaml", templateParam.Company, templateParam.Name)
	err = os.WriteFile(tplPath, tplBuffer.Bytes(), 0600)
	if err != nil {
		log.Println(err)
		return "", err
	}

	const cmdDockerCompose = "docker-compose"

	switch actionType {
	case pbAct.ActionType_UP:
		err = exec.Command(cmdDockerCompose, "-f", tplPath, "up", "-d").Run()

	case pbAct.ActionType_DOWN:
		err = exec.Command(cmdDockerCompose, "-f", tplPath, "down").Run()

	default:
		err = fmt.Errorf("unknown action type: %s", actionType)

	}
	if err != nil {
		log.Println(err)
		return "", err
	}

	time.Sleep(5 * time.Second)

	output, err := exec.Command(cmdDockerCompose, "-f", tplPath, "ps").Output()
	if err != nil {
		log.Println(err)
		return "", err
	}

	err = os.Remove(tplPath)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return string(output), err
}

func (h *Handler) sendDeployResponse(response *pbAct.ActionResponse) {
	if err := h.streamClient.PublishResponse(response); err != nil {
		log.Println(err)
		return
	}
}
