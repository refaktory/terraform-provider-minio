package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func datasourceBucket() *schema.Resource {
	return &schema.Resource{
		ReadContext:   resourceBucketRead,
		Schema: schemaBucket(),
	}
}
