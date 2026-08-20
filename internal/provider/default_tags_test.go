// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"reflect"
	"testing"
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
