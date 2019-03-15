package checker

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/iam/v1"
)

func getAllRoles(ctx context.Context, organizationResource string, projectLabels map[string]string) (map[string][]string, []cloudresourcemanager.Project, []string) {
	builtInRoles, err := getBuiltInRoles(ctx)

	orgRoles, err := getRolesForResource(ctx, organizationResource)

	if err != nil {
		log.Fatal(err)
	}

	folderNames, err := getFolderNames(organizationResource)

	validParents := StrSet{}
	validParents.Add(organizationResource)
	validParents.Add(folderNames...)

	projectRoles, projects := getAllProjectPolicies(ctx, validParents, projectLabels)

	allRoles := mergeMaps(builtInRoles, orgRoles, projectRoles)

	return allRoles, projects, folderNames
}

func getAllProjectPolicies(ctx context.Context, validParents StrSet, projectLabels map[string]string) (map[string][]string, []cloudresourcemanager.Project) {
	cloudresourcemanagerService := getResourceServiceFromContext(ctx)

	var wg sync.WaitGroup
	roleCh := make(chan map[string][]string, 16384)

	projects := []cloudresourcemanager.Project{}

	filters := buildProjectFilter(projectLabels)
	req := cloudresourcemanagerService.Projects.List().Filter(filters)
	if err := req.Pages(ctx, func(page *cloudresourcemanager.ListProjectsResponse) error {
		for _, project := range page.Projects {
			projects = append(projects, *project)
			resource := fmt.Sprintf("%ss/%s", project.Parent.Type, project.Parent.Id)
			if validParents.Contains(resource) {
				wg.Add(1)

				go getProjectRoles(ctx, *project, roleCh, &wg)
			}
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	wg.Wait()
	close(roleCh)

	roles := make(map[string][]string, 4096)
	for role := range roleCh {
		roles = mergeMaps(roles, role)
	}

	return roles, projects
}

func getBuiltInRoles(ctx context.Context) (map[string][]string, error) {
	roles := make(map[string][]string, 200)

	iamService := getIamServiceFromContext(ctx)

	req := iamService.Roles.List().View("FULL")

	if err := req.Pages(ctx, func(page *iam.ListRolesResponse) error {
		for _, role := range page.Roles {
			roles[role.Name] = role.IncludedPermissions
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	return roles, nil
}

func getRolesForResource(ctx context.Context, resource string) (map[string][]string, error) {
	roles := make(map[string][]string, 200)

	iamService := getIamServiceFromContext(ctx)

	fullResourceName := fmt.Sprintf("//cloudresourcemanager.googleapis.com/%s", resource)
	rb := &iam.QueryGrantableRolesRequest{
		FullResourceName: fullResourceName,
		View:             "FULL",
	}
	req := iamService.Roles.QueryGrantableRoles(rb)

	if err := req.Pages(ctx, func(page *iam.QueryGrantableRolesResponse) error {
		for _, role := range page.Roles {
			roles[role.Name] = role.IncludedPermissions
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	return roles, nil
}

func getProjectRoles(ctx context.Context, project cloudresourcemanager.Project, roleCh chan map[string][]string, wg *sync.WaitGroup) {
	defer wg.Done()

	resource := fmt.Sprintf("projects/%s", project.ProjectId)
	roles, err := getRolesForResource(ctx, resource)

	if err != nil {
		log.Fatal(err)
	}

	roleCh <- roles
}

// Recursively enumerates all folders in a given organisation
func getFolderNames(organizationResource string) ([]string, error) {
	ctx := context.Background()

	c, err := google.DefaultClient(ctx, cloudresourcemanagerv2.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	cloudresourcemanagerService, err := cloudresourcemanagerv2.New(c)
	if err != nil {
		return nil, err
	}

	return _getFolderNamesRecursive(ctx, organizationResource, *cloudresourcemanagerService)
}

func _getFolderNamesRecursive(ctx context.Context, resource string, cloudresourcemanagerService cloudresourcemanagerv2.Service) ([]string, error) {
	folderNames := []string{}

	req := cloudresourcemanagerService.Folders.List().Parent(resource)
	if err := req.Pages(ctx, func(page *cloudresourcemanagerv2.ListFoldersResponse) error {
		for _, folder := range page.Folders {
			folderNames = append(folderNames, folder.Name)
		}
		return nil
	}); err != nil {
		return folderNames, err
	}

	for _, folderName := range folderNames {
		childNames, err := _getFolderNamesRecursive(ctx, folderName, cloudresourcemanagerService)
		if err == nil {
			folderNames = append(folderNames, childNames...)
		}
	}

	return folderNames, nil
}

func buildProjectFilter(projectLabels map[string]string) string {
	filters := []string{"lifecycleState:ACTIVE"}
	for key, value := range projectLabels {
		filters = append(filters, fmt.Sprintf("labels.%s=%s", key, value))
	}
	return strings.Join(filters, " ")
}
