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
	keyAccessKey    = "access_key"
	keySecretKey    = "secret_key"
	keyUserPolicies = "policies"
)

func resourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			keyAccessKey: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The access key for the user. This is also the unique ID.",
				ForceNew:    true,
			},
			keySecretKey: &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The secret key for the user.",
				Sensitive:   true,
			},
			keyUserPolicies: &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "The name of the canned policy valid for this user.",
			},
		},
	}
}

func dataGetUserPolicies(data *schema.ResourceData) []string {
	policies := data.Get(keyUserPolicies).([]interface{})
	var policyStrings []string
	for _, policyRaw := range policies {
		policyStrings = append(policyStrings, policyRaw.(string))
	}
	return policyStrings
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	client := m.(*minioContext).admin
	accessKey := d.Get(keyAccessKey).(string)
	secretKey := d.Get(keySecretKey).(string)
	policies := dataGetUserPolicies(d)

	log.Printf("[DEBUG] Creating minio user: '%s'\n", accessKey)
	if err := client.AddUser(ctx, accessKey, secretKey); err != nil {
		return diag.FromErr(err)
	}

	if len(policies) > 0 {
		policyString := strings.Join(policies, ",")

		if err := client.SetPolicy(ctx, policyString, accessKey, false); err != nil {
			// TODO: the user is already created at this point, so we just add a
			// warning if the policy could not be applied.
			// Maybe we should hard-error instead and delete the user?
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Warning,
				Summary:       "Could not set policy for user: " + err.Error(),
				AttributePath: cty.GetAttrPath(keyUserPolicies),
			})
			d.Set(keyUserPolicies, nil)
		}
	}

	d.SetId(accessKey)
	return diags
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	accessKey := d.Id()
	client := m.(*minioContext).admin

	users, err := client.ListUsers(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	user, found := users[accessKey]
	if found == false {
		return diag.Errorf("User does not exist")
	}

	d.Set(keyAccessKey, accessKey)

	policies := strings.Split(user.PolicyName, ",")
	d.Set(keyUserPolicies, policies)
	// d.Set(KEY_SECRET_KEY, user.SecretKey)

	return diags
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange(keyBucketName) {
		return diag.FromErr(errors.New("Users can not be renamed"))
	}

	accessKey := d.Id()
	client := m.(*minioContext).admin

	if d.HasChange(keyUserPolicies) {
		newPolicies := dataGetUserPolicies(d)
		if len(newPolicies) == 0 {
			return diag.Errorf("Can not set policies to empty after the user has been assigned other policies. This is a minio API limitation.")
		}
		newPolicy := strings.Join(newPolicies, ",")
		if err := client.SetPolicy(ctx, newPolicy, accessKey, false); err != nil {
			return diag.Errorf("Could not change apply policy to user: %s", err)
		}
	}

	if d.HasChange(keySecretKey) {
		newSecretKey := d.Get(keySecretKey).(string)
		// TODO: implement enabled/disabled?
		if err := client.SetUser(ctx, accessKey, newSecretKey, madmin.AccountEnabled); err != nil {
			return diag.Errorf("Could nto change secret key: %s", err)
		}
	}

	return resourceUserRead(ctx, d, m)
}

func resourceUserDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	accessKey := d.Id()
	client := m.(*minioContext).admin
	if err := client.RemoveUser(ctx, accessKey); err != nil {
		return diag.FromErr(err)
	}

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")

	return diags
}
