// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/go-azure-helpers/resourcemanager/tags"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
)

func schemaDefaultTags() *pluginsdk.Schema {
	return &pluginsdk.Schema{
		Type:     pluginsdk.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &pluginsdk.Resource{
			Schema: map[string]*pluginsdk.Schema{
				"tags": {
					Type:         pluginsdk.TypeMap,
					Optional:     true,
					ValidateFunc: tags.Validate,
					Elem: &pluginsdk.Schema{
						Type: pluginsdk.TypeString,
					},
					Description: "A map of tags which is applied to every resource that supports tags.",
				},
			},
		},
	}
}

func expandDefaultTags(input []interface{}) map[string]string {
	output := make(map[string]string)

	if len(input) == 0 || input[0] == nil {
		return output
	}

	val := input[0].(map[string]interface{})

	if raw, ok := val["tags"].(map[string]interface{}); ok {
		for k, v := range raw {
			output[k] = v.(string)
		}
	}

	return output
}
