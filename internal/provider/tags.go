package provider

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func tagsIn(tags map[string]attr.Value) []awstypes.Tag {
	result := make([]awstypes.Tag, 0, len(tags))

	for k := range tags {
		v := tags[k].(types.String)
		tag := awstypes.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.ValueString()),
		}
		result = append(result, tag)
	}

	return result
}

func tagsOut(tags []awstypes.Tag) (types.Map, diag.Diagnostics) {
	tagMap := map[string]attr.Value{}

	for _, tag := range tags {
		tagMap[*tag.Key] = types.StringValue(*tag.Value)
	}
	mapVal, d := types.MapValue(types.StringType, tagMap)

	return mapVal, d
}
