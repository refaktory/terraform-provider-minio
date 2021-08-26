package provider

import (
	"context"

	"github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	keyConfigEndpoint = "endpoint"
	keyConfigSsl      = "ssl"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

// Provider -
func NewMinioProvider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"endpoint": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The Minio server domain.\nMust not include http[s]://!\nEg: my-minio.domain.com",
				Required:    true,
			},
			"ssl": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If true, https:// will be used.",
			},
			keyAccessKey: &schema.Schema{
				Type:        schema.TypeString,
				Sensitive:   true,
				Required:    true,
				Description: "The access key (username).\nShould be the minio root user or a user with sufficient permissions.",
			},
			keySecretKey: &schema.Schema{
				Type:        schema.TypeString,
				Sensitive:   true,
				Required:    true,
				Description: "The secret key (password).\nShould be the minio root user or a user with sufficient permissions.",
			},
		},
		ConfigureContextFunc: providerConfigure,
		ResourcesMap: map[string]*schema.Resource{
			"minio_bucket":        resourceBucket(),
			"minio_user":          resourceUser(),
			"minio_canned_policy": resourceCannedPolicy(),
		},
		DataSourcesMap: map[string]*schema.Resource{},
	}
}

type minioContext struct {
	api   *minio.Client
	admin *madmin.AdminClient
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	endpoint := d.Get(keyConfigEndpoint).(string)
	accessKey := d.Get(keyAccessKey).(string)
	secretKey := d.Get(keySecretKey).(string)
	ssl := false
	sslOpt := d.Get(keyConfigSsl)
	if sslOpt != nil {
		ssl = sslOpt.(bool)
	}

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	api, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: ssl,
	})
	if err != nil {
		return nil, diag.FromErr(err)
	}
	admin, err := madmin.New(endpoint, accessKey, secretKey, ssl)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	mctx := &minioContext{
		api:   api,
		admin: admin,
	}

	return mctx, diags
}
