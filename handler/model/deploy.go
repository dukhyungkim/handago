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
