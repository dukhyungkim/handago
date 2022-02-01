package template_model

import (
	pbAct "github.com/dukhyungkim/libharago/gen/go/proto/action"
)

type DeployTemplate struct {
	Name        string
	ResourceURL string
}

func NewDeployTemplate(request *pbAct.ActionRequest_DeployRequest) *DeployTemplate {
	return &DeployTemplate{
		Name:        request.GetName(),
		ResourceURL: request.GetResourceUrl(),
	}
}
