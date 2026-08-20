// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"

	helperTags "github.com/hashicorp/go-azure-helpers/resourcemanager/tags"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tags"
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
					ValidateFunc: helperTags.Validate,
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
			// Validate should have ignored this error already
			value, _ := tags.TagValueToString(v)
			output[k] = value
		}
	}

	return output
}

func mergeDefaultTags(defaultTags map[string]string, resourceTags map[string]interface{}) map[string]interface{} {
	output := make(map[string]interface{}, len(defaultTags)+len(resourceTags))
	for k, v := range defaultTags {
		output[k] = v
	}
	for k, v := range resourceTags {
		output[k] = v
	}
	return output
}

func resourceSupportsDefaultTags(resource *schema.Resource) bool {
	tagsSchema, ok := resource.Schema["tags"]
	if !ok {
		return false
	}
	if tagsSchema.Type != schema.TypeMap || !tagsSchema.Optional || tagsSchema.Computed {
		return false
	}
	elem, ok := tagsSchema.Elem.(*schema.Schema)
	return ok && elem.Type == schema.TypeString
}

func validateMergedTags(mergedTags map[string]interface{}, resourceTagCount int, defaultTagCount int) error {
	if len(mergedTags) > 50 {
		return fmt.Errorf("merged tags exceed Azure's 50-tag limit: %d resource tags + %d provider `default_tags`", resourceTagCount, defaultTagCount)
	}

	for k, v := range mergedTags {
		if len(k) > 512 {
			return fmt.Errorf("the maximum length for a tag key is 512 characters: %q is %d characters", k, len(k))
		}
		// the schema constrains tag values to strings before this runs
		if value, ok := v.(string); ok && len(value) > 256 {
			return fmt.Errorf("the maximum length for a tag value is 256 characters: the value for %q is %d characters", k, len(value))
		}
	}

	return nil
}
