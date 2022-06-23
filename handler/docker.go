package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"handago/common"
	"handago/config"
	"handago/handler/model"
	"handago/stream"
	"html/template"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	pbAct "github.com/dukhyungkim/libharago/gen/go/proto/action"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Handler struct {
	company      string
	host         string
	base         string
	adapter      string
	etcdClient   *clientv3.Client
	streamClient *stream.Client
}

func New(opts *config.Options, cfg *config.Etcd, streamClient *stream.Client) (*Handler, error) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: common.DefaultTimeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, common.ErrConnEtcd(err)
	}

	return &Handler{
		company:      opts.Company,
		host:         opts.Host,
		base:         opts.Base,
		adapter:      opts.Adapter,
		etcdClient:   etcdClient,
		streamClient: streamClient,
	}, nil
}

func (h *Handler) Close() {
	if err := h.etcdClient.Close(); err != nil {
		log.Printf("failed to close etcd client cleany; %v\n", err)
	}
}

func (h *Handler) HandleUpDownAction(request *pbAct.ActionRequest) {
	switch actionType := request.GetType(); actionType {
	case pbAct.ActionType_UP, pbAct.ActionType_DOWN:
		templateParam := model.NewDeployTemplateParam(h.host, h.base, request.GetReqDeploy())
		templateParam.SetCompany(h.company)

		if !templateParam.IsMatchAdapter(h.adapter) {
			log.Println("adapter is unmatched")
			return
		}

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

func (h *Handler) loadTemplate(name string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), common.DefaultTimeout)
	defer cancel()

	deployKey := fmt.Sprintf("/templates/%s", name)
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

	const cmdDocker = "docker"

	var output []byte
	switch actionType {
	case pbAct.ActionType_UP:
		output, err = exec.Command(cmdDocker, "compose", "-f", tplPath, "up", "-d").CombinedOutput()

	case pbAct.ActionType_DOWN:
		output, err = exec.Command(cmdDocker, "compose", "-f", tplPath, "down").CombinedOutput()

	default:
		return "", fmt.Errorf("unknown action type: %s", actionType)
	}
	if err != nil {
		errWithOutput := makeErrWithOutput(output, err)
		log.Println(errWithOutput)
		return "", errWithOutput
	}

	time.Sleep(5 * time.Second)

	output, err = exec.Command(cmdDocker, "compose", "-f", tplPath, "ps").CombinedOutput()
	if err != nil {
		errWithOutput := makeErrWithOutput(output, err)
		log.Println(errWithOutput)
		return "", errWithOutput
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

func makeErrWithOutput(output []byte, err error) error {
	if len(output) == 0 {
		return err
	}

	sb := strings.Builder{}
	sb.Write(output)
	sb.WriteString("\n")
	sb.WriteString(err.Error())
	return errors.New(sb.String())
}
