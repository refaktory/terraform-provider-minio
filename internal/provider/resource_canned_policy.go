package provider

import (
	"context"
	"encoding/json"
	"reflect"

	// "github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	keyPolicyName   = "name"
	keyPolicyPolicy = "policy"
)

func resourceCannedPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCannedPolicyCreate,
		ReadContext:   resourceCannedPolicyRead,
		// UpdateContext: resourceCannedPolicyUpdate,
		DeleteContext: resourceCannedPolicyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			keyPolicyName: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name for this policy. This is also the unique ID.",
				ForceNew:    true,
			},
			keyPolicyPolicy: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The policy definition as a map - will be encoded as JSON. See https://docs.min.io/docs/minio-multi-user-quickstart-guide.html for an example.",
				// NOTE: apparently the API does not support changing the policy config,
				// so we must re-create.
				ForceNew: true,
			},
		},
	}
}

func resourceCannedPolicyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	client := m.(*minioContext).admin
	name := d.Get(keyPolicyName).(string)
	policyJSON := d.Get(keyPolicyPolicy).(string)
	if err := client.AddCannedPolicy(ctx, name, []byte(policyJSON)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(name)
	if err := resourceCannedPolicyRead(ctx, d, m); err != nil {
		return err
	}

	return diags
}

func resourceCannedPolicyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Id()
	client := m.(*minioContext).admin

	d.Set(keyPolicyName, name)

	// Compare policy content.
	// This must be done because the policy is specified as a json STRING, and the
	// server might store the json with slighty different formatting, which
	// produces a mismatch even if the policy is the same.
	// So we decode both the server policy and the config policy to a
	// map[string]interface{} and deep-compare it.
	// Only if the server policy is different do we update the value in the
	// ResourceData
	// TODO: use Schema.DiffSuppressFunc instead!
	currentPolicyJSON, err := client.InfoCannedPolicy(ctx, name)
	if err != nil {
		return diag.FromErr(err)
	}
	var currentPolicy map[string]interface{}
	if err := json.Unmarshal(currentPolicyJSON, &currentPolicy); err != nil {
		return diag.Errorf("Could not decode JSON policy: %e", err)
	}

	originalPolicyJSON := []byte(d.Get(keyPolicyPolicy).(string))
	var originalPolicy map[string]interface{}
	if err := json.Unmarshal(originalPolicyJSON, &originalPolicy); err != nil {
		return diag.Errorf("Could not decode JSON policy: %e", err)
	}

	if !reflect.DeepEqual(currentPolicy, originalPolicy) {
		d.Set(keyPolicyPolicy, string(currentPolicyJSON))
	}

	return diags
}

func resourceCannedPolicyUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("Canned policies can not be updated")
	// if d.HasChange(POLICY_KEY_NAME) {
	// 	return diag.FromErr(errors.New("CannedPolicys can not be renamed"))
	// }
	// if d.HasChange(POLICY_KEY_POLICY) {
	// 	return diag.FromErr(errors.New("CannedPolicys can not be changed"))
	// }

	// return resourceCannedPolicyRead(ctx, d, m)
}

func resourceCannedPolicyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Id()
	client := m.(*minioContext).admin
	if err := client.RemoveCannedPolicy(ctx, name); err != nil {
		return diag.FromErr(err)
	}

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")

	return diags
}
