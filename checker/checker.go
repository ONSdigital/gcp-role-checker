package checker

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
)

type serviceKeyType int

var resourceServiceKey serviceKeyType = 1
var iamServiceKey serviceKeyType = 2

func contextWithServices(ctx context.Context, cloudresourcemanagerService *cloudresourcemanager.Service, iamService *iam.Service) context.Context {
	ctx = context.WithValue(ctx, resourceServiceKey, cloudresourcemanagerService)
	ctx = context.WithValue(ctx, iamServiceKey, iamService)
	return ctx
}

func getResourceServiceFromContext(ctx context.Context) *cloudresourcemanager.Service {
	service, ok := ctx.Value(resourceServiceKey).(*cloudresourcemanager.Service)
	if ok {
		return service
	}
	return nil
}

func getIamServiceFromContext(ctx context.Context) *iam.Service {
	service, ok := ctx.Value(iamServiceKey).(*iam.Service)
	if ok {
		return service
	}
	return nil
}

// RunChecker ...
func RunChecker(organizationResource string, projectLabels map[string]string, dataDir string) {
	ctx := context.Background()

	c, err := google.DefaultClient(ctx, cloudresourcemanager.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
	}

	cloudresourcemanagerService, err := cloudresourcemanager.New(c)
	if err != nil {
		log.Fatal(err)
	}

	iamService, err := iam.New(c)
	if err != nil {
		log.Fatal(err)
	}
	ctx = contextWithServices(ctx, cloudresourcemanagerService, iamService)

	roles, projects, folderNames := getAllRoles(ctx, organizationResource, projectLabels)

	os.Mkdir(dataDir, os.ModePerm)

	rolesBlob, ok := json.Marshal(roles)
	if ok == nil {
		ioutil.WriteFile(path.Join(dataDir, "roles.json"), rolesBlob, 0644)
	}

	allMembers := getAllBindings(ctx, organizationResource, projects, folderNames, roles)

	allMembersBlob, ok := json.Marshal(allMembers)
	if ok == nil {
		ioutil.WriteFile(path.Join(dataDir, "members.json"), allMembersBlob, 0644)
	}
}
