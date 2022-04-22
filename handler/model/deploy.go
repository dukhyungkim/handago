package model

import (
	pbAct "github.com/dukhyungkim/libharago/gen/go/proto/action"
)

type DeployTemplateParam struct {
	Company     string
	Name        string
	ResourceURL string
	Host        string
	Base        string
}

func NewDeployTemplateParam(host, base string, request *pbAct.ActionRequest_DeployRequest) *DeployTemplateParam {
	return &DeployTemplateParam{
		Company:     "Shared",
		Name:        request.GetName(),
		ResourceURL: request.GetResourceUrl(),
		Host:        host,
		Base:        base,
	}
}

func (t *DeployTemplateParam) SetCompany(company string) {
	t.Company = company
}

func (t *DeployTemplateParam) ToActionResponse(space, output string, actionType pbAct.ActionType) *pbAct.ActionResponse {
	return &pbAct.ActionResponse{
		Type:  actionType,
		Space: space,
		Response_OneOf: &pbAct.ActionResponse_RespDeploy{
			RespDeploy: &pbAct.ActionResponse_DeployResponse{
				Host:        t.Host,
				Text:        output,
				Company:     t.Company,
				ResourceUrl: t.ResourceURL,
			},
		},
	}
}
