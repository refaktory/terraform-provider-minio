package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

const (
	keyBucketLifecycleRule         = "lifecycle_rule"
	keyBucketLifecycleId           = "id"
	keyBucketLifecycleDays         = "days"
	keyBucketLifecycleDate         = "date"
	keyBucketLifecycleExpiration   = "expiration"
	keyBucketLifecycleTransition   = "transition"
	keyBucketLifecycleStorageClass = "storage_class"
	keyBucketLifecycleEnabled      = "enabled"

	lifecycleStatusEnabled  = "Enabled"
	lifecycleStatusDisabled = "Disabled"
	lifecycleDateFormat     = "2006-01-02"
)

func schemaBucketLifecycle() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				keyBucketLifecycleId: {
					Type:     schema.TypeString,
					Optional: true,
					Computed: true,
				},
				keyBucketLifecycleEnabled: {
					Type:     schema.TypeBool,
					Optional: true,
					Default:  true,
				},
				keyBucketLifecycleExpiration: {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							keyBucketLifecycleDate: {
								Type:         schema.TypeString,
								Optional:     true,
								ValidateFunc: validBucketLifecycleTimestamp,
							},
							keyBucketLifecycleDays: {
								Type:         schema.TypeInt,
								Optional:     true,
								ValidateFunc: validation.IntAtLeast(0),
							},
						},
					},
				},
				keyBucketLifecycleTransition: {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							keyBucketLifecycleDate: {
								Type:         schema.TypeString,
								Optional:     true,
								ValidateFunc: validBucketLifecycleTimestamp,
							},
							keyBucketLifecycleDays: {
								Type:         schema.TypeInt,
								Optional:     true,
								ValidateFunc: validation.IntAtLeast(0),
							},
							keyBucketLifecycleStorageClass: {
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func resourceBucketLifecycleRead(ctx context.Context, bucketName string, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*minioContext).api

	lifecycle, err := client.GetBucketLifecycle(ctx, bucketName)
	if err != nil {
		if err.Error() == "The lifecycle configuration does not exist" {
			lifecycle = nil
		} else {
			return diag.FromErr(err)
		}
	}

	lifecycleRules := make([]map[string]interface{}, 0)

	if lifecycle != nil {
		for _, lifecycleRule := range lifecycle.Rules {
			rule := make(map[string]interface{})

			if lifecycleRule.ID != "" {
				rule[keyBucketLifecycleId] = lifecycleRule.ID
			}

			rule[keyBucketLifecycleEnabled] = (lifecycleRule.Status == lifecycleStatusEnabled)

			if !lifecycleRule.Expiration.IsNull() {
				expirationList := make([]interface{}, 1)
				expiration := make(map[string]interface{})
				expirationList[0] = expiration

				if !lifecycleRule.Expiration.IsDateNull() {
					expiration[keyBucketLifecycleDate] = lifecycleRule.Expiration.Date.Format(lifecycleDateFormat)
				}

				if !lifecycleRule.Expiration.IsDaysNull() {
					expiration[keyBucketLifecycleDays] = int(lifecycleRule.Expiration.Days)
				}

				rule[keyBucketLifecycleExpiration] = expirationList
			}

			if !lifecycleRule.Transition.IsNull() {
				transitionList := make([]interface{}, 1)
				transition := make(map[string]interface{})
				transitionList[0] = transition

				if !lifecycleRule.Transition.IsDateNull() {
					transition[keyBucketLifecycleDate] = lifecycleRule.Transition.Date.Format(lifecycleDateFormat)
				}

				if !lifecycleRule.Transition.IsDaysNull() {
					transition[keyBucketLifecycleDays] = int(lifecycleRule.Transition.Days)
				}

				transition[keyBucketLifecycleStorageClass] = lifecycleRule.Transition.StorageClass

				rule[keyBucketLifecycleTransition] = transitionList
			}

			lifecycleRules = append(lifecycleRules, rule)
		}
	}

	if err := d.Set(keyBucketLifecycleRule, lifecycleRules); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceBucketLifecycleUpdate(ctx context.Context, bucketName string, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	client := m.(*minioContext).api

	config := lifecycle.NewConfiguration()
	lifecycleRules := d.Get(keyBucketLifecycleRule).([]interface{})

	if len(lifecycleRules) > 0 && lifecycleRules[0] != nil {
		for _, lifecycleRule := range lifecycleRules {
			r := lifecycleRule.(map[string]interface{})
			rule := lifecycle.Rule{}

			if val, ok := r[keyBucketLifecycleId].(string); ok && val != "" {
				rule.ID = val
			} else {
				rule.ID = resource.PrefixedUniqueId("tf-")
			}

			if val, ok := r[keyBucketLifecycleEnabled].(bool); ok && val {
				rule.Status = lifecycleStatusEnabled
			} else {
				rule.Status = lifecycleStatusDisabled
			}

			if expiration, ok := r[keyBucketLifecycleExpiration].([]interface{}); ok && len(expiration) > 0 && expiration[0] != nil {
				e := expiration[0].(map[string]interface{})

				if val, ok := e[keyBucketLifecycleDate].(string); ok && val != "" {
					if date, err := time.Parse(lifecycleDateFormat, val); err != nil {
						diags = append(diags, diag.FromErr(err)...)
					} else {
						rule.Expiration.Date.Time = date
					}
				}

				if val, ok := e[keyBucketLifecycleDays].(int); ok && val != 0 {
					rule.Expiration.Days = lifecycle.ExpirationDays(val)
				}
			}

			if transition, ok := r[keyBucketLifecycleTransition].([]interface{}); ok && len(transition) > 0 && transition[0] != nil {
				e := transition[0].(map[string]interface{})

				if val, ok := e[keyBucketLifecycleDate].(string); ok && val != "" {
					if date, err := time.Parse(lifecycleDateFormat, val); err != nil {
						diags = append(diags, diag.FromErr(err)...)
					} else {
						rule.Transition.Date.Time = date
					}
				}

				if val, ok := e[keyBucketLifecycleDays].(int); ok && val != 0 {
					rule.Transition.Days = lifecycle.ExpirationDays(val)
				}

				if val, ok := e[keyBucketLifecycleStorageClass].(string); ok && val != "" {
					rule.Transition.StorageClass = val
				}
			}

			config.Rules = append(config.Rules, rule)
		}
	}

	if diags.HasError() {
		return diags
	}

	if err := client.SetBucketLifecycle(ctx, bucketName, config); err != nil {
		return diag.FromErr(err)
	}

	return append(diags, resourceBucketLifecycleRead(ctx, bucketName, d, m)...)
}

func validBucketLifecycleTimestamp(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if _, err := time.Parse(lifecycleDateFormat, value); err != nil {
		errors = append(errors, fmt.Errorf("%q does not have a valid date format", value))
	}

	return
}
