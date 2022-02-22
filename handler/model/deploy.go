package model

import (
	pbAct "github.com/dukhyungkim/libharago/gen/go/proto/action"
)

type SharedDeployTemplate struct {
	Name        string
	ResourceURL string
	Host        string
	Base        string
}

func NewSharedDeployTemplate(host, base string, request *pbAct.ActionRequest_DeployRequest) *SharedDeployTemplate {
	return &SharedDeployTemplate{
		Name:        request.GetName(),
		ResourceURL: request.GetResourceUrl(),
		Host:        host,
		Base:        base,
	}
}

func (t *SharedDeployTemplate) ToActionResponse(space, output string) *pbAct.ActionResponse {
	return &pbAct.ActionResponse{
		Type:  pbAct.ActionType_DEPLOY,
		Space: space,
		Response_OneOf: &pbAct.ActionResponse_RespDeploy{
			RespDeploy: &pbAct.ActionResponse_DeployResponse{
				Host:        t.Host,
				Text:        output,
				Company:     "Shared",
				ResourceUrl: t.ResourceURL,
			},
		},
	}
}

type CompanyDeployTemplate struct {
	Company string
	*SharedDeployTemplate
}

func NewCompanyDeployTemplate(company, host, base string, request *pbAct.ActionRequest_DeployRequest) *CompanyDeployTemplate {
	return &CompanyDeployTemplate{
		Company:              company,
		SharedDeployTemplate: NewSharedDeployTemplate(host, base, request),
	}
}

func (t *CompanyDeployTemplate) ToActionResponse(space, output string) *pbAct.ActionResponse {
	return &pbAct.ActionResponse{
		Type:  pbAct.ActionType_DEPLOY,
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
