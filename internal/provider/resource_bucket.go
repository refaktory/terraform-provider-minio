package provider

import (
	"context"
	"errors"

	// Minio client SDK
	"github.com/minio/minio-go/v7"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	keyBucketName              = "name"
	keyBucketVersioningEnabled = "versioning_enabled"
)

func schemaBucket() objectSchema {
	return map[string]*schema.Schema{
		keyBucketName: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The name of the bucket. Can not be changed without recreating.",
			ForceNew:    true,
		},
		keyBucketVersioningEnabled: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Enables versioning. Note: this is only available if the Minio server is run with erasure codes enabled. See https://docs.min.io/docs/minio-erasure-code-quickstart-guide",
		},
		keyBucketLifecycleRule: schemaBucketLifecycle(),
	}
}

func resourceBucket() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBucketCreate,
		ReadContext:   resourceBucketRead,
		UpdateContext: resourceBucketUpdate,
		DeleteContext: resourceBucketDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: schemaBucket(),
	}
}

func resourceBucketCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	client := m.(*minioContext).api
	name := d.Get(keyBucketName).(string)

	if err := client.MakeBucket(ctx, name, minio.MakeBucketOptions{}); err != nil {
		return diag.FromErr(err)
	}

	// TODO: check if server supports versioning before trying this?
	versioningEnabled := d.Get(keyBucketVersioningEnabled).(bool)
	if versioningEnabled {
		if err := client.EnableVersioning(ctx, name); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity:      diag.Warning,
				Summary:       "Could not enable versioning: " + err.Error(),
				AttributePath: cty.GetAttrPath(keyBucketVersioningEnabled),
			})
		}
	}

	diags = append(diags, resourceBucketLifecycleUpdate(ctx, name, d, m)...)
	if diags.HasError() {
		return diags
	}

	d.SetId(name)
	return diags
}

func resourceBucketRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics
	var name string

	if d.Id() == "" {
		name = d.Get(keyBucketName).(string)
		d.SetId(name)
	} else {
		name = d.Id()
		if err := d.Set(keyBucketName, name); err != nil {
			return diag.FromErr(err)
		}
	}

	client := m.(*minioContext).api

	// Ensure that bucket exists.
	flag, err := client.BucketExists(ctx, name)
	if err != nil {
		return diag.FromErr(err)
	}
	if !flag {
		return diag.FromErr(errors.New("Bucket " + name + " does not exist"))
	}
	if err := d.Set(keyBucketName, name); err != nil {
		return diag.FromErr(err)
	}

	// Check versioning.
	versionConfig, err := client.GetBucketVersioning(ctx, name)
	if err != nil {
		return diag.FromErr(err)
	}
	versioningEnabled := versionConfig.Enabled()
	if err := d.Set(keyBucketVersioningEnabled, versioningEnabled); err != nil {
		return diag.FromErr(err)
	}

	diags = append(diags, resourceBucketLifecycleRead(ctx, name, d, m)...)
	if diags.HasError() {
		return diags
	}

	return diags
}

func resourceBucketUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if d.HasChange(keyBucketName) {
		return diag.FromErr(errors.New("Buckets can not be renamed"))
	}

	name := d.Id()
	client := m.(*minioContext).api

	if d.HasChange(keyBucketVersioningEnabled) {
		// TODO: check if server supports versioning before trying this?
		enabled := d.Get(keyBucketVersioningEnabled).(bool)
		if enabled {
			if err := client.EnableVersioning(ctx, name); err != nil {
				return []diag.Diagnostic{{
					Severity:      diag.Warning,
					Summary:       "Could not enable versioning: " + err.Error(),
					AttributePath: cty.GetAttrPath(keyBucketVersioningEnabled),
				}}
			}
		} else {
			if err := client.SuspendVersioning(ctx, name); err != nil {
				return []diag.Diagnostic{{
					Severity:      diag.Warning,
					Summary:       "Could not disable versioning: " + err.Error(),
					AttributePath: cty.GetAttrPath(keyBucketVersioningEnabled),
				}}
			}
		}
	}

	if d.HasChange(keyBucketLifecycleRule) {
		diags := resourceBucketLifecycleUpdate(ctx, name, d, m)
		if diags.HasError() {
			return diags
		}
	}

	return resourceBucketRead(ctx, d, m)
}

func resourceBucketDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// TODO: buckets can only be deleted if they are empty we could manually delete all files in the bucket to enable this.

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	name := d.Id()
	client := m.(*minioContext).api

	if err := client.RemoveBucket(ctx, name); err != nil {
		return diag.FromErr(err)
	}

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")

	return diags
}
