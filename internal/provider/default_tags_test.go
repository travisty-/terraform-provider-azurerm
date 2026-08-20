// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestExpandDefaultTags(t *testing.T) {
	testData := []struct {
		Name     string
		Input    []interface{}
		Expected map[string]string
	}{
		{
			Name:     "Absent Block",
			Input:    []interface{}{},
			Expected: map[string]string{},
		},
		{
			Name: "Empty Block",
			Input: []interface{}{
				map[string]interface{}{},
			},
			Expected: map[string]string{},
		},
		{
			Name: "Tags Set",
			Input: []interface{}{
				map[string]interface{}{
					"tags": map[string]interface{}{
						"environment": "prod",
						"team":        "platform",
					},
				},
			},
			Expected: map[string]string{
				"environment": "prod",
				"team":        "platform",
			},
		},
		{
			Name: "Non-String Tag Value",
			Input: []interface{}{
				map[string]interface{}{
					"tags": map[string]interface{}{
						"retry": 3,
					},
				},
			},
			Expected: map[string]string{
				"retry": "3",
			},
		},
	}

	for _, testCase := range testData {
		t.Logf("[DEBUG] Test Case: %q..", testCase.Name)
		result := expandDefaultTags(testCase.Input)
		if !reflect.DeepEqual(result, testCase.Expected) {
			t.Fatalf("expected %+v but got %+v", testCase.Expected, result)
		}
	}
}

func TestMergeDefaultTags(t *testing.T) {
	testData := []struct {
		Name         string
		DefaultTags  map[string]string
		ResourceTags map[string]interface{}
		Expected     map[string]interface{}
	}{
		{
			Name:         "Both Empty",
			DefaultTags:  map[string]string{},
			ResourceTags: map[string]interface{}{},
			Expected:     map[string]interface{}{},
		},
		{
			Name:         "Defaults Only",
			DefaultTags:  map[string]string{"environment": "prod"},
			ResourceTags: map[string]interface{}{},
			Expected:     map[string]interface{}{"environment": "prod"},
		},
		{
			Name:         "Disjoint",
			DefaultTags:  map[string]string{"environment": "prod"},
			ResourceTags: map[string]interface{}{"cost_center": "msft"},
			Expected:     map[string]interface{}{"environment": "prod", "cost_center": "msft"},
		},
		{
			Name:         "Resource Tags Take Precedence",
			DefaultTags:  map[string]string{"environment": "prod"},
			ResourceTags: map[string]interface{}{"environment": "dev"},
			Expected:     map[string]interface{}{"environment": "dev"},
		},
	}

	for _, testCase := range testData {
		t.Logf("[DEBUG] Test Case: %q..", testCase.Name)
		result := mergeDefaultTags(testCase.DefaultTags, testCase.ResourceTags)
		if !reflect.DeepEqual(result, testCase.Expected) {
			t.Fatalf("expected %+v but got %+v", testCase.Expected, result)
		}
	}
}

func TestResourceSupportsDefaultTags(t *testing.T) {
	testData := []struct {
		Name     string
		Resource *schema.Resource
		Expected bool
	}{
		{
			Name:     "No Tags",
			Resource: &schema.Resource{Schema: map[string]*schema.Schema{}},
			Expected: false,
		},
		{
			Name: "Standard Tags",
			Resource: &schema.Resource{Schema: map[string]*schema.Schema{
				"tags": commonschema.Tags(),
			}},
			Expected: true,
		},
		{
			Name: "ForceNew Tags",
			Resource: &schema.Resource{Schema: map[string]*schema.Schema{
				"tags": commonschema.TagsForceNew(),
			}},
			Expected: true,
		},
		{
			Name: "Computed Tags",
			Resource: &schema.Resource{Schema: map[string]*schema.Schema{
				"tags": {
					Type:     schema.TypeMap,
					Optional: true,
					Computed: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			}},
			Expected: false,
		},
		{
			Name: "List Of Strings",
			Resource: &schema.Resource{Schema: map[string]*schema.Schema{
				"tags": {
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			}},
			Expected: false,
		},
	}

	for _, testCase := range testData {
		t.Logf("[DEBUG] Test Case: %q..", testCase.Name)
		if result := resourceSupportsDefaultTags(testCase.Resource); result != testCase.Expected {
			t.Fatalf("expected %t but got %t", testCase.Expected, result)
		}
	}
}

func TestValidateMergedTags(t *testing.T) {
	overLimit := make(map[string]interface{})
	for i := 0; i < 51; i++ {
		overLimit[fmt.Sprintf("key%d", i)] = "value"
	}

	if err := validateMergedTags(overLimit, 31, 20); err == nil {
		t.Fatalf("expected an error for >50 merged tags but got none")
	} else if !strings.Contains(err.Error(), "50") {
		t.Fatalf("expected the error to mention the 50-tag limit, got: %s", err)
	}

	longValue := map[string]interface{}{"key": strings.Repeat("a", 257)}
	if err := validateMergedTags(longValue, 0, 1); err == nil {
		t.Fatalf("expected an error for a 257-character value but got none")
	}

	if err := validateMergedTags(map[string]interface{}{"environment": "prod"}, 0, 1); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}
