//go:build unit
// +build unit

package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
)

type mocks int

// Create the mock.
func (mocks) NewResource(args pulumi.MockResourceArgs) (string, resource.PropertyMap, error) {
	outputs := args.Inputs.Mappable()
	return args.Name + "_id", resource.NewPropertyMapFromMap(outputs), nil
}

func (mocks) Call(args pulumi.MockCallArgs) (resource.PropertyMap, error) {
	outputs := map[string]interface{}{}
	return resource.NewPropertyMapFromMap(outputs), nil
}

// Applying unit tests.
func TestInfrastructure(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		infra, err := createInfrastructure(ctx)
		if err != nil {
			fmt.Print("Couldn't create infra")
		}
		assert.NoError(t, err)
		var wg sync.WaitGroup
		wg.Add(1)

        // TODO(check 1): Resources have lower-case names (only [a-z-])
		for _, res := range infra.DdbTableItems {
			pulumi.All(res.ID()).ApplyT(func(all []interface{}) error {
				id := fmt.Sprintf("%v", all[0].(pulumi.ID))
				assert.Equal(
					t,
					id,
					strings.ToLower(id),
					fmt.Sprintf(
						"Resource name '%s' should be lower-case.",
						id,
					),
				)
				return nil
			})
		}
		
		for _, res := range(infra.DdbTables) {
			pulumi.All(res.ID()).ApplyT(func(all []interface{}) error {
				id := fmt.Sprintf("%v", all[0].(pulumi.ID))
				assert.Equal(
					t,
					id,
					strings.ToLower(id),
					fmt.Sprintf(
						"Resource name '%s' should be lower-case.",
						id,
					),
				)
				return nil
			})
		}
		
		for _, res := range(infra.Lambdas) {
			pulumi.All(res.ID()).ApplyT(func(all []interface{}) error {
				id := fmt.Sprintf("%v", all[0].(pulumi.ID))
				assert.Equal(
					t,
					id,
					strings.ToLower(id),
					fmt.Sprintf(
						"Resource name '%s' should be lower-case.",
						id,
					),
				)
				return nil
			})
		}
		
		// There seem to be complications validating the names of resources
		// created using the 'pulumi-aws-apigateway' Crosswalk module.
		//
		// for _, res := range(infra.RestApis) {
		// 	pulumi.All(res.Api.RootResourceId()).ApplyT(func(all []interface{}) error {
		// 		id := fmt.Sprintf("%v", all[0].(pulumi.StringOutput))
		// 		assert.Equal(
		// 			t,
		// 			id,
		// 			strings.ToLower(id),
		// 			fmt.Sprintf(
		// 				"Resource name '%s' should be lower-case.",
		// 				id,
		// 			),
		// 		)
		// 		return nil
		// 	})
		// }
		
		for _, res := range(infra.S3Buckets) {
			pulumi.All(res.ID()).ApplyT(func(all []interface{}) error {
				id := fmt.Sprintf("%v", all[0].(pulumi.ID))
				assert.Equal(
					t,
					id,
					strings.ToLower(id),
					fmt.Sprintf(
						"Resource name '%s' should be lower-case.",
						id,
					),
				)
				return nil
			})
		}
		
		for _, res := range(infra.S3Objects) {
			pulumi.All(res.ID()).ApplyT(func(all []interface{}) error {
				id := fmt.Sprintf("%v", all[0].(pulumi.ID))
				assert.Equal(
					t,
					id,
					strings.ToLower(id),
					fmt.Sprintf(
						"Resource name '%s' should be lower-case.",
						id,
					),
				)
				return nil
			})
		}
		wg.Done()

        // TODO(check 2): Check the count of resources created.
		
        // TODO(check 3): All resources must have an owner tag.

		// Test if the service has tags and a name tag.
		// pulumi.All(infra.URN(), infra.server.Tags).ApplyT(func(all []interface{}) error {
		// 	urn := all[0].(pulumi.URN)
		// 	tags := all[1].(map[string]string)

		// 	assert.Containsf(t, tags, "Name", "missing a Name tag on server %v", urn)
		// 	wg.Done()
		// 	return nil
		// })

		// // Test if the instance is configured with user_data.
		// pulumi.All(infra.server.URN(), infra.server.UserData).ApplyT(func(all []interface{}) error {
		// 	urn := all[0].(pulumi.URN)
		// 	userData := all[1].(*string)

		// 	assert.Nilf(t, userData, "illegal use of userData on server %v", urn)
		// 	wg.Done()
		// 	return nil
		// })

		// // Test if port 22 for ssh is exposed.
		// pulumi.All(infra.group.URN(), infra.group.Ingress).ApplyT(func(all []interface{}) error {
		// 	urn := all[0].(pulumi.URN)
		// 	ingress := all[1].([]ec2.SecurityGroupIngress)

		// 	for _, i := range ingress {
		// 		openToInternet := false
		// 		for _, b := range i.CidrBlocks {
		// 			if b == "0.0.0.0/0" {
		// 				openToInternet = true
		// 				break
		// 			}
		// 		}

		// 		assert.Falsef(t, i.FromPort == 22 && openToInternet, "illegal SSH port 22 open to the Internet (CIDR 0.0.0.0/0) on group %v", urn)
		// 	}

		// 	wg.Done()
		// 	return nil
		// })

		wg.Wait()
		return nil
	}, pulumi.WithMocks("project", "stack", mocks(0)))
	assert.NoError(t, err)
}
