package provider

import (
	"context"
	"errors"
	"fmt"
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
	keyUserGroups   = "groups"
)

func schemaUser() objectSchema {
	return map[string]*schema.Schema{
		keyAccessKey: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The access key for the user. This is also the unique ID.",
			ForceNew:    true,
		},
		keySecretKey: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The secret key for the user.",
			Sensitive:   true,
		},
		keyUserPolicies: {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Optional:    true,
			Description: "The names of the canned policies valid for this user.",
		},
		keyUserGroups: {
			Type: schema.TypeList,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Optional:    true,
			Description: "The names of the groups this user belongs to.",
		},
	}
}

func resourceUser() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceUserCreate,
		ReadContext:   resourceUserRead,
		UpdateContext: resourceUserUpdate,
		DeleteContext: resourceUserDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: schemaUser(),
	}
}

func datasourceUser() *schema.Resource {
	s := schemaUser()
	s[keySecretKey].Required = false
	s[keySecretKey].Optional = true
	return &schema.Resource{
		ReadContext: resourceUserRead,
		Schema:      s,
	}
}

func dataGetUserPolicies(data *schema.ResourceData) []string {
	return dataGetStringList(data, keyUserPolicies)
}

func dataGetUserGroups(data *schema.ResourceData) []string {
	return dataGetStringList(data, keyUserGroups)
}

func stringSliceContains(slice []string, value string) bool {
	for _, item := range slice {
		if value == item {
			return true
		}
	}
	return false
}

// Compare two string slices.
// Returns the removed and the added values as two separate slices.
func stringSliceDiff(old []string, _new []string) ([]string, []string) {
	var added []string
	var removed []string

	for _, oldValue := range old {
		if !stringSliceContains(_new, oldValue) {
			removed = append(removed, oldValue)
		}
	}
	for _, newValue := range _new {
		if !stringSliceContains(old, newValue) {
			added = append(added, newValue)
		}
	}

	return added, removed
}

func stringSliceRemove(slice []string, value string) []string {
	var newSlice []string

	for _, oldValue := range slice {
		if oldValue != value {
			newSlice = append(newSlice, oldValue)
		}
	}

	return newSlice
}

// Check if all the specified groups exist on a Minio server.
// Returns an error if any of the groups do not exist, or nil otherwise.
func verifyGroupsExist(ctx context.Context, client *madmin.AdminClient, groups []string) error {
	existingGroups, err := client.ListGroups(ctx)
	var missing []string
	if err != nil {
		return err
	}
	for _, group := range groups {
		if !stringSliceContains(existingGroups, group) {
			missing = append(missing, group)
		}
	}

	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("Group(s) do not exist: %s", strings.Join(missing, ", "))
}

func resourceUserCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	client := m.(*minioContext).admin
	accessKey := d.Get(keyAccessKey).(string)
	secretKey := d.Get(keySecretKey).(string)
	policies := dataGetUserPolicies(d)
	groups := dataGetUserGroups(d)

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
			if err := d.Set(keyUserPolicies, nil); err != nil {
				return diag.FromErr(err)
			}
		}
	}
	if len(groups) > 0 {
		if err := verifyGroupsExist(ctx, client, groups); err != nil {
			return []diag.Diagnostic{{
				Severity:      diag.Error,
				Summary:       err.Error(),
				AttributePath: cty.GetAttrPath(keyUserGroups),
			}}
		}

		var actuallyAddedGroups []string

		for _, group := range groups {
			err := client.UpdateGroupMembers(ctx, madmin.GroupAddRemove{
				Group:    group,
				Members:  []string{accessKey},
				IsRemove: false,
			})

			if err != nil {
				diags = append(diags, diag.Diagnostic{
					Severity:      diag.Warning,
					Summary:       "Could not add user to group " + group + ": " + err.Error(),
					AttributePath: cty.GetAttrPath(keyUserGroups),
				})
			} else {
				actuallyAddedGroups = append(actuallyAddedGroups, group)
			}
		}
		if err := d.Set(keyUserGroups, actuallyAddedGroups); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(accessKey)
	return diags
}

func resourceUserRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	accessKey := d.Get(keyAccessKey).(string)
	client := m.(*minioContext).admin

	if d.Id() == "" {
		d.SetId(accessKey)
	}

	users, err := client.ListUsers(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	user, found := users[accessKey]
	if found == false {
		return diag.Errorf("User does not exist")
	}

	if err := d.Set(keyAccessKey, accessKey); err != nil {
		return diag.FromErr(err)
	}

	policies := strings.Split(user.PolicyName, ",")
	if err := d.Set(keyUserPolicies, policies); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(keyUserGroups, user.MemberOf); err != nil {
		return diag.FromErr(err)
	}

	// TODO: how to handle this? API seems to not return the key.
	// d.Set(KEY_SECRET_KEY, user.SecretKey)

	return diags
}

func resourceUserUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange(keyAccessKey) {
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

	if d.HasChange(keyUserGroups) {
		oldRaw, newRaw := d.GetChange(keyUserGroups)
		old := interfaceToStringSlice(oldRaw.([]interface{}))
		_new := interfaceToStringSlice(newRaw.([]interface{}))
		added, removed := stringSliceDiff(old, _new)

		if err := verifyGroupsExist(ctx, client, added); err != nil {
			return []diag.Diagnostic{{
				Severity:      diag.Error,
				AttributePath: cty.GetAttrPath(keyUserGroups),
				Summary:       "Invalid group(s): " + err.Error(),
			}}
		}

		for _, newGroup := range added {
			err := client.UpdateGroupMembers(ctx, madmin.GroupAddRemove{
				Group:    newGroup,
				Members:  []string{accessKey},
				IsRemove: false,
			})
			if err != nil {
				if err := d.Set(keyUserGroups, old); err != nil {
					return diag.FromErr(err)
				}
				return []diag.Diagnostic{{
					Severity:      diag.Error,
					AttributePath: cty.GetAttrPath(keyUserGroups),
					Summary:       "Could not add user to group " + newGroup + ": " + err.Error(),
				}}
			}
			old = append(old, newGroup)
		}

		for _, removedGroup := range removed {
			err := client.UpdateGroupMembers(ctx, madmin.GroupAddRemove{
				Group:    removedGroup,
				Members:  []string{accessKey},
				IsRemove: true,
			})
			if err != nil {
				if err := d.Set(keyUserGroups, old); err != nil {
					return diag.FromErr(err)
				}
				return []diag.Diagnostic{{
					Severity:      diag.Error,
					AttributePath: cty.GetAttrPath(keyUserGroups),
					Summary:       "Could not remove user from group " + removedGroup + ": " + err.Error(),
				}}
			}
			old = stringSliceRemove(old, removedGroup)
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
