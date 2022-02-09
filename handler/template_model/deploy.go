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

type CompanyDeployTemplate struct {
	Company string
	Host    string
	*DeployTemplate
}

func NewCompanyDeployTemplate(company string, host string, request *pbAct.ActionRequest_DeployRequest) *CompanyDeployTemplate {
	return &CompanyDeployTemplate{
		Company:        company,
		Host:           host,
		DeployTemplate: NewDeployTemplate(request),
	}
}
