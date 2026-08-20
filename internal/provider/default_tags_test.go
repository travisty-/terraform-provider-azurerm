// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-cty/cty"
	ctymsgpack "github.com/hashicorp/go-cty/cty/msgpack"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
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

func testDefaultTagsProvider(defaultTags map[string]string) *schema.Provider {
	resources := map[string]*schema.Resource{
		"azurerm_default_tags_test": {
			Schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Required: true,
				},
				"tags": commonschema.Tags(),
			},
		},
	}
	addDefaultTagsSupport(resources)
	p := &schema.Provider{ResourcesMap: resources}
	p.SetMeta(&clients.Client{DefaultTags: defaultTags})
	return p
}

// testForceNewDefaultTagsProvider mirrors testDefaultTagsProvider but with a ForceNew
// `tags` schema, matching resources such as azurerm_automation_runtime_environment.
func testForceNewDefaultTagsProvider(defaultTags map[string]string) *schema.Provider {
	resources := map[string]*schema.Resource{
		"azurerm_default_tags_forcenew_test": {
			Schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Required: true,
				},
				"tags": commonschema.TagsForceNew(),
			},
		},
	}
	addDefaultTagsSupport(resources)
	p := &schema.Provider{ResourcesMap: resources}
	p.SetMeta(&clients.Client{DefaultTags: defaultTags})
	return p
}

// testObjectWithOverrides builds a cty object of the given type with every attribute
// null except the provided overrides
func testObjectWithOverrides(ty cty.Type, overrides map[string]cty.Value) cty.Value {
	attrs := make(map[string]cty.Value)
	for name, attrType := range ty.AttributeTypes() {
		if v, ok := overrides[name]; ok {
			attrs[name] = v
			continue
		}
		attrs[name] = cty.NullVal(attrType)
	}
	return cty.ObjectVal(attrs)
}

func encodeDynamicValue(t *testing.T, v cty.Value, ty cty.Type) *tfprotov5.DynamicValue {
	t.Helper()
	mp, err := ctymsgpack.Marshal(v, ty)
	if err != nil {
		t.Fatalf("marshalling value: %+v", err)
	}
	return &tfprotov5.DynamicValue{MsgPack: mp}
}

func decodeDynamicValue(t *testing.T, v *tfprotov5.DynamicValue, ty cty.Type) cty.Value {
	t.Helper()
	val, err := ctymsgpack.Unmarshal(v.MsgPack, ty)
	if err != nil {
		t.Fatalf("unmarshalling value: %+v", err)
	}
	return val
}

func planDefaultTagsResourceChange(t *testing.T, p *schema.Provider, resourceType string, prior, proposed, config cty.Value) *tfprotov5.PlanResourceChangeResponse {
	t.Helper()

	r, ok := p.ResourcesMap[resourceType]
	if !ok {
		t.Fatalf("resource %q was not found in the provider", resourceType)
	}
	ty := r.CoreConfigSchema().ImpliedType()

	server := schema.NewGRPCProviderServer(p)
	resp, err := server.PlanResourceChange(context.Background(), &tfprotov5.PlanResourceChangeRequest{
		TypeName:         resourceType,
		PriorState:       encodeDynamicValue(t, prior, ty),
		ProposedNewState: encodeDynamicValue(t, proposed, ty),
		Config:           encodeDynamicValue(t, config, ty),
	})
	if err != nil {
		t.Fatalf("planning change: %+v", err)
	}
	return resp
}

func TestDefaultTags_planScenarios(t *testing.T) {
	defaults := map[string]string{"environment": "prod", "team": "platform"}

	p := testDefaultTagsProvider(defaults)
	r := p.ResourcesMap["azurerm_default_tags_test"]
	ty := r.CoreConfigSchema().ImpliedType()

	name := cty.StringVal("test")

	tags := func(kv map[string]string) cty.Value {
		vals := make(map[string]cty.Value)
		for k, v := range kv {
			vals[k] = cty.StringVal(v)
		}
		return cty.MapVal(vals)
	}
	noTags := cty.NullVal(cty.Map(cty.String))
	unknownTags := cty.UnknownVal(cty.Map(cty.String))
	obj := func(id, tagsVal cty.Value) cty.Value {
		return testObjectWithOverrides(ty, map[string]cty.Value{"id": id, "name": name, "tags": tagsVal})
	}
	nullID := cty.NullVal(cty.String)
	testID := cty.StringVal("test-id")

	plannedTags := func(t *testing.T, resp *tfprotov5.PlanResourceChangeResponse) cty.Value {
		t.Helper()
		for _, d := range resp.Diagnostics {
			if d.Severity == tfprotov5.DiagnosticSeverityError {
				t.Fatalf("unexpected error diagnostic: %s - %s", d.Summary, d.Detail)
			}
		}
		return decodeDynamicValue(t, resp.PlannedState, ty).GetAttr("tags")
	}

	t.Run("create with no resource tags applies the defaults", func(t *testing.T) {
		resp := planDefaultTagsResourceChange(t, p, "azurerm_default_tags_test",
			cty.NullVal(ty), obj(nullID, unknownTags), obj(nullID, noTags))
		if result := plannedTags(t, resp); !result.RawEquals(tags(map[string]string{"environment": "prod", "team": "platform"})) {
			t.Fatalf("unexpected planned tags: %#v", result)
		}
	})

	t.Run("resource tags win on collision", func(t *testing.T) {
		configured := tags(map[string]string{"environment": "dev", "cost_center": "msft"})
		resp := planDefaultTagsResourceChange(t, p, "azurerm_default_tags_test",
			cty.NullVal(ty), obj(nullID, configured), obj(nullID, configured))
		expected := tags(map[string]string{"environment": "dev", "cost_center": "msft", "team": "platform"})
		if result := plannedTags(t, resp); !result.RawEquals(expected) {
			t.Fatalf("unexpected planned tags: %#v", result)
		}
	})

	t.Run("no diff when state already matches the merged tags", func(t *testing.T) {
		configured := tags(map[string]string{"cost_center": "msft"})
		state := tags(map[string]string{"cost_center": "msft", "environment": "prod", "team": "platform"})
		resp := planDefaultTagsResourceChange(t, p, "azurerm_default_tags_test",
			obj(testID, state), obj(testID, configured), obj(testID, configured))
		if result := plannedTags(t, resp); !result.RawEquals(state) {
			t.Fatalf("expected planned tags to equal state, got: %#v", result)
		}
	})

	t.Run("drift on a default tag is corrected", func(t *testing.T) {
		configured := tags(map[string]string{"cost_center": "msft"})
		drifted := tags(map[string]string{"cost_center": "msft", "environment": "hacked", "team": "platform"})
		resp := planDefaultTagsResourceChange(t, p, "azurerm_default_tags_test",
			obj(testID, drifted), obj(testID, configured), obj(testID, configured))
		expected := tags(map[string]string{"cost_center": "msft", "environment": "prod", "team": "platform"})
		if result := plannedTags(t, resp); !result.RawEquals(expected) {
			t.Fatalf("unexpected planned tags: %#v", result)
		}
	})

	t.Run("unknown resource tags defer the merge to apply time", func(t *testing.T) {
		resp := planDefaultTagsResourceChange(t, p, "azurerm_default_tags_test",
			cty.NullVal(ty), obj(nullID, unknownTags), obj(nullID, unknownTags))
		if result := decodeDynamicValue(t, resp.PlannedState, ty).GetAttr("tags"); result.IsKnown() {
			t.Fatalf("expected planned tags to be unknown, got: %#v", result)
		}
	})

	t.Run("ignore_changes keeps resource-level edits frozen", func(t *testing.T) {
		// core substitutes the prior state into the proposed value for ignored attributes,
		// so config and proposed disagree; the proposed value must win as the merge base
		configured := tags(map[string]string{"environment": "new"})
		priorTags := tags(map[string]string{"environment": "old", "team": "platform"})
		resp := planDefaultTagsResourceChange(t, p, "azurerm_default_tags_test",
			obj(testID, priorTags), obj(testID, tags(map[string]string{"environment": "old"})), obj(testID, configured))
		expected := tags(map[string]string{"environment": "old", "team": "platform"})
		if result := plannedTags(t, resp); !result.RawEquals(expected) {
			t.Fatalf("unexpected planned tags: %#v", result)
		}
	})

	t.Run("over 50 merged tags is an error", func(t *testing.T) {
		many := make(map[string]string, 49)
		for i := 0; i < 49; i++ {
			many[fmt.Sprintf("key%d", i)] = "value"
		}
		configured := tags(many)
		resp := planDefaultTagsResourceChange(t, p, "azurerm_default_tags_test",
			cty.NullVal(ty), obj(nullID, configured), obj(nullID, configured))
		found := false
		for _, d := range resp.Diagnostics {
			if d.Severity == tfprotov5.DiagnosticSeverityError && strings.Contains(d.Summary+d.Detail, "50-tag limit") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a 50-tag limit error diagnostic, got: %+v", resp.Diagnostics)
		}
	})

	t.Run("without defaults behavior is unchanged", func(t *testing.T) {
		noDefaults := testDefaultTagsProvider(map[string]string{})
		configured := tags(map[string]string{"cost_center": "msft"})
		resp := planDefaultTagsResourceChange(t, noDefaults, "azurerm_default_tags_test",
			cty.NullVal(ty), obj(nullID, configured), obj(nullID, configured))
		if result := plannedTags(t, resp); !result.RawEquals(configured) {
			t.Fatalf("unexpected planned tags: %#v", result)
		}
	})

	t.Run("without defaults removing tags from config still plans a removal", func(t *testing.T) {
		// with the schema flipped to Optional+Computed the SDK proposes the prior value
		// when config is unset; the diff must plan the removal regardless
		noDefaults := testDefaultTagsProvider(map[string]string{})
		state := tags(map[string]string{"cost_center": "msft"})
		resp := planDefaultTagsResourceChange(t, noDefaults, "azurerm_default_tags_test",
			obj(testID, state), obj(testID, state), obj(testID, noTags))
		if result := plannedTags(t, resp); !result.RawEquals(cty.MapValEmpty(cty.String)) {
			t.Fatalf("expected planned tags to be empty, got: %#v", result)
		}
	})

	t.Run("without defaults and no tags in config plans no tags", func(t *testing.T) {
		noDefaults := testDefaultTagsProvider(map[string]string{})
		resp := planDefaultTagsResourceChange(t, noDefaults, "azurerm_default_tags_test",
			cty.NullVal(ty), obj(nullID, unknownTags), obj(nullID, noTags))
		result := decodeDynamicValue(t, resp.PlannedState, ty).GetAttr("tags")
		// A freshly-created resource has no prior state to compare the merged (here empty)
		// tags against, so the diff can't be cleared to a known no-op the way an update can.
		// SetUnknowns (grpc_provider.go) then unconditionally promotes the null Computed
		// `tags` to unknown for any create plan; this is confined to create, since an
		// update can always compare against its actual prior state.
		if result.IsKnown() {
			t.Fatalf("expected planned tags to be unknown, got: %#v", result)
		}
	})

	t.Run("without defaults and null prior tags is a true no-op on update", func(t *testing.T) {
		noDefaults := testDefaultTagsProvider(map[string]string{})
		prior := obj(testID, noTags)
		resp := planDefaultTagsResourceChange(t, noDefaults, "azurerm_default_tags_test",
			prior, obj(testID, noTags), obj(testID, noTags))
		for _, d := range resp.Diagnostics {
			if d.Severity == tfprotov5.DiagnosticSeverityError {
				t.Fatalf("unexpected error diagnostic: %s - %s", d.Summary, d.Detail)
			}
		}
		planned := decodeDynamicValue(t, resp.PlannedState, ty)
		if !planned.RawEquals(prior) {
			t.Fatalf("expected the plan to be a no-op matching prior state, got: %#v", planned)
		}
	})

	t.Run("removing a default plans its removal", func(t *testing.T) {
		fewer := testDefaultTagsProvider(map[string]string{"environment": "prod"})
		configured := tags(map[string]string{"cost_center": "msft"})
		state := tags(map[string]string{"cost_center": "msft", "environment": "prod", "team": "platform"})
		resp := planDefaultTagsResourceChange(t, fewer, "azurerm_default_tags_test",
			obj(testID, state), obj(testID, configured), obj(testID, configured))
		expected := tags(map[string]string{"cost_center": "msft", "environment": "prod"})
		if result := plannedTags(t, resp); !result.RawEquals(expected) {
			t.Fatalf("unexpected planned tags: %#v", result)
		}
	})
}

func TestDefaultTags_forceNewPlanScenarios(t *testing.T) {
	name := cty.StringVal("test")
	noTags := cty.NullVal(cty.Map(cty.String))
	testID := cty.StringVal("test-id")

	t.Run("without defaults and null prior tags plans no replacement", func(t *testing.T) {
		noDefaults := testForceNewDefaultTagsProvider(map[string]string{})
		ty := noDefaults.ResourcesMap["azurerm_default_tags_forcenew_test"].CoreConfigSchema().ImpliedType()
		obj := func(id, tagsVal cty.Value) cty.Value {
			return testObjectWithOverrides(ty, map[string]cty.Value{"id": id, "name": name, "tags": tagsVal})
		}
		prior := obj(testID, noTags)
		resp := planDefaultTagsResourceChange(t, noDefaults, "azurerm_default_tags_forcenew_test",
			prior, obj(testID, noTags), obj(testID, noTags))
		for _, d := range resp.Diagnostics {
			if d.Severity == tfprotov5.DiagnosticSeverityError {
				t.Fatalf("unexpected error diagnostic: %s - %s", d.Summary, d.Detail)
			}
		}
		if len(resp.RequiresReplace) > 0 {
			t.Fatalf("expected no forced replacement, got RequiresReplace: %v", resp.RequiresReplace)
		}
		planned := decodeDynamicValue(t, resp.PlannedState, ty)
		if !planned.RawEquals(prior) {
			t.Fatalf("expected the plan to be a no-op matching prior state, got: %#v", planned)
		}
	})

	t.Run("newly configured defaults plan a replacement", func(t *testing.T) {
		p := testForceNewDefaultTagsProvider(map[string]string{"environment": "prod"})
		ty := p.ResourcesMap["azurerm_default_tags_forcenew_test"].CoreConfigSchema().ImpliedType()
		obj := func(id, tagsVal cty.Value) cty.Value {
			return testObjectWithOverrides(ty, map[string]cty.Value{"id": id, "name": name, "tags": tagsVal})
		}
		state := cty.MapVal(map[string]cty.Value{"cost_center": cty.StringVal("msft")})
		resp := planDefaultTagsResourceChange(t, p, "azurerm_default_tags_forcenew_test",
			obj(testID, state), obj(testID, state), obj(testID, state))
		tagsPath := tftypes.NewAttributePath().WithAttributeName("tags")
		found := false
		for _, attr := range resp.RequiresReplace {
			if attr.Equal(tagsPath) {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected RequiresReplace to include tags, got: %v", resp.RequiresReplace)
		}
	})
}
