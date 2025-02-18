package provider

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func SetFrameworkTags(state *types.Map, tags []awstypes.Tag, emptyTags bool) {
	if len(tags) == 0 {
		if emptyTags {
			emptyMap, _ := types.MapValue(types.StringType, map[string]attr.Value{})
			*state = emptyMap
		}
		return
	}

	mapVal, d := tagsOut(tags)
	if d != nil {
		return
	}

	*state = mapVal
}

func SetFrameworkFromString(state *types.String, value string, emptyAsNull bool) {
	if emptyAsNull {
		if state.IsNull() &&
			value != "" {
			*state = types.StringValue(value)
		}
	} else {
		*state = types.StringValue(value)
	}
}

func SetFrameworkFromtStringPointer(state *types.String, value *string) {
	if value != nil {
		strVal := aws.ToString(value)
		*state = types.StringValue(strVal)
	}
}

func SetFrameworkFromBool(state *types.Bool, value bool) {
	*state = types.BoolValue(value)
}

func SetFrameworkFromOutputLocationModel(state *[]OutputLocationModel, value *awstypes.InstanceAssociationOutputLocation) {
	if value == nil {
		return
	}

	if value.S3Location == nil {
		return
	}

	var outputLocations []OutputLocationModel
	outputLocation := &OutputLocationModel{}
	SetFrameworkFromtStringPointer(&outputLocation.S3Region, value.S3Location.OutputS3Region)
	SetFrameworkFromtStringPointer(&outputLocation.S3BucketName, value.S3Location.OutputS3BucketName)
	SetFrameworkFromtStringPointer(&outputLocation.S3KeyPrefix, value.S3Location.OutputS3KeyPrefix)
	*state = append(outputLocations, *outputLocation)
}

func targetsIn(targetList types.List) []awstypes.Target {
	var targets []awstypes.Target

	for _, targetElem := range targetList.Elements() {
		targetObj := targetElem.(types.Object)
		attributes := targetObj.Attributes()

		target := awstypes.Target{}

		for k := range attributes {
			if k == "key" {
				value := attributes[k].(types.String)
				target.Key = value.ValueStringPointer()
			}
			if k == "values" {
				values := attributes[k].(types.List)
				for _, v := range values.Elements() {
					strval := v.(types.String)
					target.Values = append(target.Values, strval.ValueString())
				}
			}
		}

		targets = append(targets, target)
	}

	return targets
}

func targetsOut(ctx context.Context, targetsOutput []awstypes.Target) types.List {
	var values []attr.Value

	for _, t := range targetsOutput {
		v, _ := types.ListValueFrom(ctx, types.StringType, t.Values)
		elements, _ := types.ListValue(types.StringType, v.Elements())

		objVal, _ := types.ObjectValue(
			map[string]attr.Type{
				"key":    types.StringType,
				"values": types.ListType{ElemType: types.StringType},
			},
			map[string]attr.Value{
				"key":    types.StringValue(aws.ToString(t.Key)),
				"values": elements,
			},
		)

		values = append(values, objVal)
	}

	targetsList, _ := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"key":    types.StringType,
				"values": types.ListType{ElemType: types.StringType},
			},
		},
		values,
	)

	return targetsList
}

func parametersIn(ctx context.Context, parameters map[string]attr.Value) map[string][]string {
	inputParameters := make(map[string][]string)

	for k := range parameters {
		v := parameters[k].(types.List)
		var values []string
		_ = v.ElementsAs(ctx, &values, false)
		inputParameters[k] = values
	}

	return inputParameters
}

func parametersOut(input map[string][]string) types.Map {
	attrMap := make(map[string]attr.Value, len(input))

	for key, values := range input {
		listAttrValues := make([]attr.Value, len(values))
		for i, v := range values {
			listAttrValues[i] = types.StringValue(v)
		}

		listValue, err := types.ListValue(types.StringType, listAttrValues)
		if err != nil {
			return types.Map{}
		}

		attrMap[key] = listValue
	}
	mapVal, _ := types.MapValue(types.ListType{ElemType: types.StringType}, attrMap)

	return mapVal
}
