package provider

import (
	"context"
	"errors"
	"log"
	"strings"

	// Minio ADMIN client SDK
	// The admin client allows user management, which is not exposed in the
	// regular SDK.
	"github.com/minio/madmin-go"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	keyGroupName     = "name"
	keyGroupPolicies = "policies"
)

func schemaGroup() objectSchema {
	return map[string]*schema.Schema{
		keyGroupName: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The name for the group.",
			ForceNew:    true,
		},
		keyGroupPolicies: {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Optional:    true,
			Description: "The policies assigned to this group.",
		},
	}
}

func resourceGroup() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceGroupCreate,
		ReadContext:   resourceGroupRead,
		UpdateContext: resourceGroupUpdate,
		DeleteContext: resourceGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: schemaGroup(),
	}
}

func datasourceGroup() *schema.Resource {
	return &schema.Resource{
		ReadContext: resourceGroupRead,
		Schema:      schemaGroup(),
	}
}

func dataGetGroupPolicies(data *schema.ResourceData) []string {
	policies := data.Get(keyGroupPolicies).([]interface{})
	var policyStrings []string
	for _, policyRaw := range policies {
		policyStrings = append(policyStrings, policyRaw.(string))
	}
	return policyStrings
}

func resourceGroupCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	client := m.(*minioContext).admin
	groupName := d.Get(keyGroupName).(string)
	policies := dataGetGroupPolicies(d)

	log.Printf("[DEBUG] Creating minio group: '%s'\n", groupName)
	err := client.UpdateGroupMembers(ctx, madmin.GroupAddRemove{
		Group:    groupName,
		IsRemove: false,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	if len(policies) > 0 {
		policyString := strings.Join(policies, ",")

		if err := client.SetPolicy(ctx, policyString, groupName, true); err != nil {
			// TODO: the group is already created at this point, so we just add a
			// warning if the policy could not be applied.
			// Maybe we should hard-error instead and delete the group?
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Warning,
				Summary:       "Could not set policy for user: " + err.Error(),
				AttributePath: cty.GetAttrPath(keyGroupPolicies),
			})
			if err := d.Set(keyGroupPolicies, nil); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	d.SetId(groupName)
	return diags
}

func resourceGroupRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	groupName := d.Get(keyGroupName).(string)
	if d.Id() == "" {
		d.SetId(groupName)
	}
	client := m.(*minioContext).admin

	info, err := client.GetGroupDescription(ctx, groupName)
	if err != nil {
		return diag.Errorf("Could not load group %s: %e", groupName, err)
	}

	if err := d.Set(keyGroupName, groupName); err != nil {
		return diag.FromErr(err)
	}

	// Policies.
	var policies []interface{}
	for _, part := range strings.Split(info.Policy, ",") {
		if part != "" {
			policies = append(policies, &part)
		}
	}
	if err := d.Set(keyGroupPolicies, policies); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceGroupUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange(keyGroupName) {
		return diag.FromErr(errors.New("Groups can not be renamed"))
	}

	groupName := d.Id()
	client := m.(*minioContext).admin

	if d.HasChange(keyGroupPolicies) {
		newPolicies := dataGetGroupPolicies(d)
		// FIXME: handle removing old policies!
		if len(newPolicies) == 0 {
			return []diag.Diagnostic{{
				Severity:      diag.Error,
				Summary:       "Can not set policies to empty after the group has been assigned other policies. This is a minio API limitation.",
				AttributePath: cty.GetAttrPath(keyGroupPolicies),
			}}
		}
		policiesJoined := strings.Join(newPolicies, ",")
		if err := client.SetPolicy(ctx, policiesJoined, groupName, true); err != nil {
			return diag.Errorf("Could not change group policies: %s", err)
		}
	}

	return resourceGroupRead(ctx, d, m)
}

func resourceGroupDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	groupName := d.Id()
	client := m.(*minioContext).admin
	err := client.UpdateGroupMembers(ctx, madmin.GroupAddRemove{
		Group:    groupName,
		IsRemove: true,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")

	return diags
}
