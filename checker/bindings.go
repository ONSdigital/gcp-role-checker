package checker

import (
	"fmt"
	"log"
	"sync"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudresourcemanager/v1"
	cloudresourcemanagerv2 "google.golang.org/api/cloudresourcemanager/v2"
)

// Role ...
type Role struct {
	Name            string `json:"name"`
	PermissionCount int    `json:"permission_count"`
}

// Resource ...
type Resource struct {
	Name  string `json:"name"`
	Roles []Role `json:"roles"`
}

// Member ...
type Member struct {
	Resources []Resource `json:"resources"`
}

// Merge ...
func (member *Member) Merge(fromMember Member) {
	member.Resources = append(member.Resources, fromMember.Resources...)
}

// AddResourceRole ...
func (member *Member) AddResourceRole(resourceName string, role string, allRoles map[string][]string) {
	permissionCount := 0
	permissions, permOk := allRoles[role]
	if permOk {
		permissionCount = len(permissions)
	}

	resourceOk, resource := member.GetResourceByName(resourceName)

	resource.Roles = append(resource.Roles, Role{Name: role, PermissionCount: permissionCount})
	if !resourceOk {
		member.Resources = append(member.Resources, *resource)
	}
}

// GetResourceByName ...
func (member *Member) GetResourceByName(resourceName string) (bool, *Resource) {
	for _, resource := range member.Resources {
		if resource.Name == resourceName {
			return true, &resource
		}
	}
	return false, &Resource{Name: resourceName}
}

func getAllBindings(ctx context.Context, organizationResource string, projects []cloudresourcemanager.Project, folderNames []string, allRoles map[string][]string) map[string]Member {
	orgBindings := getOrgBindings(ctx, organizationResource, allRoles)
	projectBindings := getProjectBindings(ctx, projects, allRoles)
	folderBindings := getFolderBindings(folderNames, allRoles)

	return mergeRoleMaps(map[string]Member{}, orgBindings, projectBindings, folderBindings)
}

func getProjectBindings(ctx context.Context, projects []cloudresourcemanager.Project, allRoles map[string][]string) map[string]Member {
	cloudresourcemanagerService := getResourceServiceFromContext(ctx)

	ch := make(chan map[string]Member, 16384)

	var wg sync.WaitGroup
	wg.Add(len(projects))
	for _, project := range projects {
		go func(project cloudresourcemanager.Project) {
			rb := &cloudresourcemanager.GetIamPolicyRequest{}

			resp, err := cloudresourcemanagerService.Projects.GetIamPolicy(project.ProjectId, rb).Context(ctx).Do()
			if err != nil {
				log.Printf("Error on project: %v", project.Name)
				log.Fatal(err)
			}

			resourceName := fmt.Sprintf("projects/%s", project.ProjectId)
			memberRoleMap := processPolicyResponse(resourceName, *resp, allRoles)

			ch <- memberRoleMap

			wg.Done()
		}(project)
	}

	wg.Wait()
	close(ch)

	allMembers := map[string]Member{}
	for memberRoleMap := range ch {
		allMembers = mergeRoleMaps(allMembers, memberRoleMap)
	}

	return allMembers
}

func getOrgBindings(ctx context.Context, resource string, allRoles map[string][]string) map[string]Member {
	cloudresourcemanagerService := getResourceServiceFromContext(ctx)
	rb := &cloudresourcemanager.GetIamPolicyRequest{}

	resp, err := cloudresourcemanagerService.Organizations.GetIamPolicy(resource, rb).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	return processPolicyResponse(resource, *resp, allRoles)
}

func getFolderBindings(folderNames []string, allRoles map[string][]string) map[string]Member {
	ctx := context.Background()

	c, err := google.DefaultClient(ctx, cloudresourcemanagerv2.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
	}

	cloudresourcemanagerService, err := cloudresourcemanagerv2.New(c)
	if err != nil {
		log.Fatal(err)
	}

	ch := make(chan map[string]Member, 16384)

	var wg sync.WaitGroup
	wg.Add(len(folderNames))

	for _, folderName := range folderNames {
		go func(folderName string) {
			rb := &cloudresourcemanagerv2.GetIamPolicyRequest{}
			resp, err := cloudresourcemanagerService.Folders.GetIamPolicy(folderName, rb).Context(ctx).Do()
			if err != nil {
				log.Fatal(err)
			}

			memberMap := processPolicyV2Response(folderName, *resp, allRoles)

			ch <- memberMap

			wg.Done()
		}(folderName)
	}
	wg.Wait()
	close(ch)

	allMembers := map[string]Member{}
	for memberMap := range ch {
		allMembers = mergeRoleMaps(allMembers, memberMap)
	}

	return allMembers
}

func mergeRoleMaps(toMembers map[string]Member, fromMembers ...map[string]Member) map[string]Member {
	for _, fromMember := range fromMembers {
		for memberEmail, fromMember := range fromMember {
			toMember, ok := toMembers[memberEmail]
			if !ok {
				toMember = Member{}
			}
			toMember.Merge(fromMember)
			toMembers[memberEmail] = toMember
		}
	}

	return toMembers
}

func processPolicyResponse(resourceName string, policy cloudresourcemanager.Policy, allRoles map[string][]string) map[string]Member {
	memberMap := map[string]Member{}
	for _, binding := range policy.Bindings {
		for _, memberEmail := range binding.Members {
			member, ok := memberMap[memberEmail]
			if !ok {
				member = Member{}
			}
			member.AddResourceRole(resourceName, binding.Role, allRoles)
			memberMap[memberEmail] = member
		}
	}

	return memberMap
}

// TODO: there must be a better way to do this
func processPolicyV2Response(resourceName string, policy cloudresourcemanagerv2.Policy, allRoles map[string][]string) map[string]Member {
	memberMap := map[string]Member{}
	for _, binding := range policy.Bindings {
		for _, memberEmail := range binding.Members {
			member, ok := memberMap[memberEmail]
			if !ok {
				member = Member{}
			}
			member.AddResourceRole(resourceName, binding.Role, allRoles)
			memberMap[memberEmail] = member
		}
	}

	return memberMap
}
