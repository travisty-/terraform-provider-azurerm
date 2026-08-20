// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"reflect"

	helperTags "github.com/hashicorp/go-azure-helpers/resourcemanager/tags"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
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

// addDefaultTagsSupport wires provider-level default tags into every Resource with the
// standard ARM tags schema. The `tags` schema becomes Computed so its planned value can
// legitimately include the provider `default_tags`, and a CustomizeDiff merges them into
// the plan; resource-level tags take precedence. Resources read the merged value via
// d.Get("tags") during Create/Update, so no per-resource changes are needed.
func addDefaultTagsSupport(resources map[string]*schema.Resource) {
	for _, resource := range resources {
		if !resourceSupportsDefaultTags(resource) {
			continue
		}

		resource.Schema["tags"].Computed = true

		if existing := resource.CustomizeDiff; existing != nil {
			resource.CustomizeDiff = pluginsdk.CustomDiffInSequence(existing, defaultTagsCustomizeDiff)
		} else {
			resource.CustomizeDiff = defaultTagsCustomizeDiff
		}
	}
}

func defaultTagsCustomizeDiff(ctx context.Context, d *pluginsdk.ResourceDiff, meta interface{}) error {
	client, ok := meta.(*clients.Client)
	if !ok || client == nil {
		return nil
	}

	rawConfig := d.GetRawConfig()
	if rawConfig.IsNull() || !rawConfig.Type().HasAttribute("tags") {
		return nil
	}

	// The raw config distinguishes `tags` being unset from being set. When set, the
	// proposed value is the base for the merge: under `ignore_changes` core proposes the
	// prior state rather than the config, and either way the proposed value holds the
	// resource-level tags, which take precedence over the provider `default_tags`.
	base := make(map[string]interface{})
	// NewValueKnown is unreliable for TypeMap: an unresolved computed map reads back
	// as known-and-empty rather than unknown. The raw planned value distinguishes them.
	plannedIsKnown := d.GetRawPlan().GetAttr("tags").IsKnown()
	if !rawConfig.GetAttr("tags").IsNull() {
		if !plannedIsKnown {
			// the resource's tags contain values unknown until apply time - the merge
			// happens when this runs again during the apply step
			return nil
		}
		if planned, ok := d.Get("tags").(map[string]interface{}); ok {
			base = planned
		}
	}

	merged := mergeDefaultTags(client.DefaultTags, base)

	if len(client.DefaultTags) > 0 {
		if err := validateMergedTags(merged, len(base), len(client.DefaultTags)); err != nil {
			return err
		}
	}

	// A Computed TypeMap with no prior state entry is marked computed by the SDK's diff
	// engine whenever both the old and new lengths are zero, regardless of what SetNew
	// requests. Comparing against the prior state directly, rather than the already-diffed
	// planned value, and clearing the diff on a match avoids leaving that spurious entry in
	// place, which would otherwise surface as a no-op resource update or, on a ForceNew
	// `tags` schema, a spurious forced replacement.
	if plannedIsKnown && reflect.DeepEqual(rawStateTags(d), merged) {
		return d.Clear("tags")
	}

	return d.SetNew("tags", merged)
}

// rawStateTags reads `tags` directly from the resource's raw prior state, sidestepping
// the ResourceDiff's merged reader. A fresh create has an entirely null raw state, and an
// untagged resource has a null `tags` value; both read back as no tags.
func rawStateTags(d *pluginsdk.ResourceDiff) map[string]interface{} {
	rawState := d.GetRawState()
	if rawState.IsNull() {
		return map[string]interface{}{}
	}

	tagsVal := rawState.GetAttr("tags")
	if tagsVal.IsNull() || !tagsVal.IsKnown() {
		return map[string]interface{}{}
	}

	out := make(map[string]interface{}, tagsVal.LengthInt())
	for k, v := range tagsVal.AsValueMap() {
		out[k] = v.AsString()
	}
	return out
}
