package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"time"
)

var _ resource.Resource = &AWSSSMAssociationResource{}
var _ resource.ResourceWithConfigure = &AWSSSMAssociationResource{}
var _ resource.ResourceWithImportState = &AWSSSMAssociationResource{}

type AWSSSMAssociationResource struct {
	Meta Meta
}

type AWSSSMAssociationResourceModel struct {
	ApplyOnlyAtCronInterval       types.Bool            `tfsdk:"apply_only_at_cron_interval"`
	Arn                           types.String          `tfsdk:"arn"`
	AssociationId                 types.String          `tfsdk:"association_id"`
	AssociationName               types.String          `tfsdk:"association_name"`
	AssociationVersion            types.String          `tfsdk:"association_version"`
	AutomationTargetParameterName types.String          `tfsdk:"automation_target_parameter_name"`
	ComplianceSeverity            types.String          `tfsdk:"compliance_severity"`
	DocumentVersion               types.String          `tfsdk:"document_version"`
	ID                            types.String          `tfsdk:"id"`
	MaxConcurrency                types.String          `tfsdk:"max_concurrency"`
	MaxErrors                     types.String          `tfsdk:"max_errors"`
	Name                          types.String          `tfsdk:"name"`
	OutputLocation                []OutputLocationModel `tfsdk:"output_location"`
	Parameters                    types.Map             `tfsdk:"parameters"`
	ScheduleExpression            types.String          `tfsdk:"schedule_expression"`
	SyncCompliance                types.String          `tfsdk:"sync_compliance"`
	Tags                          types.Map             `tfsdk:"tags"`
	TagsAll                       types.Map             `tfsdk:"tags_all"`
	Targets                       types.List            `tfsdk:"targets"`
	WaitForSuccessTimeoutSeconds  types.Int32           `tfsdk:"wait_for_success_timeout_seconds"`
}

type OutputLocationModel struct {
	S3BucketName types.String `tfsdk:"s3_bucket_name"`
	S3KeyPrefix  types.String `tfsdk:"s3_key_prefix"`
	S3Region     types.String `tfsdk:"s3_region"`
}

func newAWSSSMAssociationResource() resource.Resource {
	return &AWSSSMAssociationResource{}
}

func (a *AWSSSMAssociationResource) Configure(_ context.Context, request resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	a.Meta = request.ProviderData.(Meta)
}

func (a *AWSSSMAssociationResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_aws_ssm_association"
}

func (a *AWSSSMAssociationResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"apply_only_at_cron_interval": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"arn": schema.StringAttribute{
				Computed: true,
			},
			"association_id": schema.StringAttribute{
				Computed: true,
			},
			"association_name": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.All(
						stringvalidator.LengthBetween(3, 128),
						stringvalidator.RegexMatches(regexache.MustCompile(`^[0-9A-Za-z_.-]{3,128}$`), "must contain only alphanumeric, underscore, hyphen, or period characters"),
					),
				},
			},
			"association_version": schema.StringAttribute{
				Computed: true,
			},
			"automation_target_parameter_name": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 50),
				},
			},
			"compliance_severity": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						[]string{
							string(awstypes.AssociationComplianceSeverityCritical),
							string(awstypes.AssociationComplianceSeverityHigh),
							string(awstypes.AssociationComplianceSeverityMedium),
							string(awstypes.AssociationComplianceSeverityLow),
							string(awstypes.AssociationComplianceSeverityUnspecified),
						}...,
					),
				},
			},
			"document_version": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexache.MustCompile(`^([$]LATEST|[$]DEFAULT|^[1-9][0-9]*$)$`), ""),
				},
			},
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"max_concurrency": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexache.MustCompile(`^([1-9][0-9]*|[1-9][0-9]%|[1-9]%|100%)$`), "must be a valid number (e.g. 10) or percentage including the percent sign (e.g. 10%)"),
				},
			},
			"max_errors": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexache.MustCompile(`^([1-9][0-9]*|[0]|[1-9][0-9]%|[0-9]%|100%)$`), "must be a valid number (e.g. 10) or percentage including the percent sign (e.g. 10%)"),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parameters": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.ListType{ElemType: types.StringType},
			},
			"schedule_expression": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
				},
			},
			"sync_compliance": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						[]string{
							string(awstypes.AssociationSyncComplianceAuto),
							string(awstypes.AssociationSyncComplianceManual),
						}...,
					),
				},
			},
			"tags": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"tags_all": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"targets": schema.ListAttribute{
				Optional: true,
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"key":    types.StringType,
						"values": types.ListType{ElemType: types.StringType},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(5),
				},
			},
			"wait_for_success_timeout_seconds": schema.Int32Attribute{
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"output_location": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"s3_bucket_name": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(3, 63),
							},
						},
						"s3_key_prefix": schema.StringAttribute{
							Optional: true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(0, 500),
							},
						},
						"s3_region": schema.StringAttribute{
							Optional: true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(3, 20),
							},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
		},
	}
}

func (a *AWSSSMAssociationResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data AWSSSMAssociationResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	input := &ssm.CreateAssociationInput{
		Name: aws.String(data.Name.ValueString()),
		Tags: tagsIn(data.Tags.Elements()),
	}

	if !data.ApplyOnlyAtCronInterval.IsNull() {
		input.ApplyOnlyAtCronInterval = data.ApplyOnlyAtCronInterval.ValueBool()
	}

	if !data.AssociationName.IsNull() {
		input.AssociationName = data.AssociationName.ValueStringPointer()
	}

	if !data.AutomationTargetParameterName.IsNull() {
		input.AutomationTargetParameterName = data.AutomationTargetParameterName.ValueStringPointer()
	}

	if !data.ComplianceSeverity.IsNull() {
		input.ComplianceSeverity = awstypes.AssociationComplianceSeverity(data.ComplianceSeverity.ValueString())
	}

	if data.DocumentVersion.ValueString() != "" {
		input.DocumentVersion = data.DocumentVersion.ValueStringPointer()
	}

	if !data.MaxConcurrency.IsNull() {
		input.MaxConcurrency = data.MaxConcurrency.ValueStringPointer()
	}

	if !data.MaxErrors.IsNull() {
		input.MaxErrors = data.MaxErrors.ValueStringPointer()
	}

	if !data.Name.IsNull() {
		input.Name = data.Name.ValueStringPointer()
	}

	if data.OutputLocation != nil {
		s3OutputLocation := &awstypes.S3OutputLocation{
			OutputS3BucketName: data.OutputLocation[0].S3BucketName.ValueStringPointer(),
		}

		if !data.OutputLocation[0].S3KeyPrefix.IsNull() {
			s3OutputLocation.OutputS3KeyPrefix = data.OutputLocation[0].S3KeyPrefix.ValueStringPointer()
		}

		if !data.OutputLocation[0].S3Region.IsNull() {
			s3OutputLocation.OutputS3Region = data.OutputLocation[0].S3Region.ValueStringPointer()
		}

		input.OutputLocation = &awstypes.InstanceAssociationOutputLocation{
			S3Location: s3OutputLocation,
		}
	}

	if !data.Parameters.IsNull() {
		input.Parameters = parametersIn(ctx, data.Parameters.Elements())
	}

	if !data.ScheduleExpression.IsNull() {
		input.ScheduleExpression = data.ScheduleExpression.ValueStringPointer()
	}

	if !data.SyncCompliance.IsNull() {
		input.SyncCompliance = awstypes.AssociationSyncCompliance(data.SyncCompliance.ValueString())
	}

	if !data.Targets.IsUnknown() && !data.Targets.IsNull() {
		input.Targets = targetsIn(data.Targets)
	}

	ssmClient := a.Meta.AWSClient.SSMClient
	output, err := ssmClient.CreateAssociation(ctx, input)
	if err != nil {
		response.Diagnostics.AddError("Error creating SSM association", err.Error())
		return
	}

	SetFrameworkTags(&data.Tags, input.Tags, false)

	SetFrameworkFromString(&data.ComplianceSeverity, string(output.AssociationDescription.ComplianceSeverity), true)
	SetFrameworkFromString(&data.SyncCompliance, string(output.AssociationDescription.SyncCompliance), true)

	// computed
	amazonResourceName := arn.ARN{
		Partition: a.Meta.AWSClient.Partition,
		Service:   "ssm",
		Region:    a.Meta.AWSClient.Region,
		AccountID: a.Meta.AWSClient.AccountID,
		Resource:  "association/" + aws.ToString(output.AssociationDescription.AssociationId),
	}.String()

	SetFrameworkFromString(&data.Arn, amazonResourceName, false)
	SetFrameworkFromtStringPointer(&data.AssociationId, output.AssociationDescription.AssociationId)
	SetFrameworkFromtStringPointer(&data.AssociationName, output.AssociationDescription.AssociationName)
	SetFrameworkFromtStringPointer(&data.AssociationVersion, output.AssociationDescription.AssociationVersion)
	SetFrameworkFromtStringPointer(&data.DocumentVersion, output.AssociationDescription.DocumentVersion)
	SetFrameworkFromtStringPointer(&data.ID, output.AssociationDescription.AssociationId)
	SetFrameworkTags(&data.TagsAll, input.Tags, true)
	data.Targets = targetsOut(ctx, output.AssociationDescription.Targets)

	if data.Parameters.IsNull() || data.Parameters.IsUnknown() {
		data.Parameters = parametersOut(output.AssociationDescription.Parameters)
	}

	if !data.WaitForSuccessTimeoutSeconds.IsNull() &&
		!data.WaitForSuccessTimeoutSeconds.IsUnknown() {
		timeout := time.Duration(data.WaitForSuccessTimeoutSeconds.ValueInt32()) * time.Second
		associationId := aws.ToString(output.AssociationDescription.AssociationId)
		if _, err := waitAssociationCreated(ctx, ssmClient, associationId, timeout, response.Diagnostics); err != nil {
			response.Diagnostics.AddError("Error creating SSM association", fmt.Sprintf("waiting for SSM Association (%s) create: %s", associationId, err.Error()))
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (a *AWSSSMAssociationResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data AWSSSMAssociationResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	association, err := FindAssociationByID(ctx, a.Meta.AWSClient.SSMClient, data.ID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error reading association", err.Error())
		return
	}
	tags, err := findAssociationTagsByID(ctx, a.Meta.AWSClient.SSMClient, data.ID.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error reading association tags", err.Error())
	}

	SetFrameworkTags(&data.Tags, tags, false)

	SetFrameworkFromString(&data.ComplianceSeverity, string(association.ComplianceSeverity), true)
	SetFrameworkFromString(&data.SyncCompliance, string(association.SyncCompliance), true)

	SetFrameworkFromtStringPointer(&data.AssociationName, association.AssociationName)
	SetFrameworkFromtStringPointer(&data.AutomationTargetParameterName, association.AutomationTargetParameterName)
	SetFrameworkFromtStringPointer(&data.MaxConcurrency, association.MaxConcurrency)
	SetFrameworkFromtStringPointer(&data.MaxErrors, association.MaxErrors)
	SetFrameworkFromtStringPointer(&data.Name, association.Name)
	SetFrameworkFromtStringPointer(&data.ScheduleExpression, association.ScheduleExpression)

	SetFrameworkFromOutputLocationModel(&data.OutputLocation, association.OutputLocation)

	// computed
	amazonResourceName := arn.ARN{
		Partition: a.Meta.AWSClient.Partition,
		Service:   "ssm",
		Region:    a.Meta.AWSClient.Region,
		AccountID: a.Meta.AWSClient.AccountID,
		Resource:  "association/" + aws.ToString(association.AssociationId),
	}.String()

	SetFrameworkFromBool(&data.ApplyOnlyAtCronInterval, association.ApplyOnlyAtCronInterval)
	SetFrameworkFromString(&data.Arn, amazonResourceName, false)
	SetFrameworkFromtStringPointer(&data.AssociationId, association.AssociationId)
	SetFrameworkFromtStringPointer(&data.AssociationVersion, association.AssociationVersion)
	SetFrameworkFromtStringPointer(&data.DocumentVersion, association.DocumentVersion)
	data.Parameters = parametersOut(association.Parameters)
	SetFrameworkTags(&data.TagsAll, tags, true)
	data.Targets = targetsOut(ctx, association.Targets)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (a *AWSSSMAssociationResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state AWSSSMAssociationResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)

	if response.Diagnostics.HasError() {
		return
	}

	input := &ssm.UpdateAssociationInput{
		AssociationId: state.AssociationId.ValueStringPointer(),
	}

	if !plan.ApplyOnlyAtCronInterval.IsNull() {
		input.ApplyOnlyAtCronInterval = plan.ApplyOnlyAtCronInterval.ValueBool()
	}

	if !plan.AssociationName.IsNull() {
		input.AssociationName = plan.AssociationName.ValueStringPointer()
	}

	if !plan.AutomationTargetParameterName.IsNull() {
		input.AutomationTargetParameterName = plan.AutomationTargetParameterName.ValueStringPointer()
	}

	if !plan.ComplianceSeverity.IsNull() {
		input.ComplianceSeverity = awstypes.AssociationComplianceSeverity(plan.ComplianceSeverity.ValueString())
	}

	if !plan.DocumentVersion.IsNull() &&
		plan.DocumentVersion.ValueString() != "" {
		input.DocumentVersion = plan.DocumentVersion.ValueStringPointer()
	} else {
		plan.DocumentVersion = state.DocumentVersion
	}

	if !plan.MaxConcurrency.IsNull() {
		input.MaxConcurrency = plan.MaxConcurrency.ValueStringPointer()
	}

	if !plan.MaxErrors.IsNull() {
		input.MaxErrors = plan.MaxErrors.ValueStringPointer()
	}

	if !plan.MaxErrors.IsNull() {
		input.MaxErrors = plan.MaxErrors.ValueStringPointer()
	}

	if plan.OutputLocation != nil {
		s3OutputLocation := &awstypes.S3OutputLocation{
			OutputS3BucketName: plan.OutputLocation[0].S3BucketName.ValueStringPointer(),
		}

		if !plan.OutputLocation[0].S3KeyPrefix.IsNull() {
			s3OutputLocation.OutputS3KeyPrefix = plan.OutputLocation[0].S3KeyPrefix.ValueStringPointer()
		}

		if !plan.OutputLocation[0].S3Region.IsNull() {
			s3OutputLocation.OutputS3Region = plan.OutputLocation[0].S3Region.ValueStringPointer()
		}

		input.OutputLocation = &awstypes.InstanceAssociationOutputLocation{
			S3Location: s3OutputLocation,
		}
	}

	if !plan.Parameters.IsNull() {
		input.Parameters = parametersIn(ctx, plan.Parameters.Elements())
	}

	if !plan.ScheduleExpression.IsNull() {
		input.ScheduleExpression = plan.ScheduleExpression.ValueStringPointer()
	}

	if !plan.SyncCompliance.IsNull() {
		input.SyncCompliance = awstypes.AssociationSyncCompliance(plan.SyncCompliance.ValueString())
	}

	stateTargets := targetsIn(state.Targets)
	if !isAutoSSMTarget(stateTargets) {
		input.Targets = targetsIn(plan.Targets)
	}

	output, err := a.Meta.AWSClient.SSMClient.UpdateAssociation(ctx, input)
	if err != nil {
		response.Diagnostics.AddError("Error updating association", err.Error())
		return
	}
	if output != nil {
		amazonResourceName := arn.ARN{
			Partition: a.Meta.AWSClient.Partition,
			Service:   "ssm",
			Region:    a.Meta.AWSClient.Region,
			AccountID: a.Meta.AWSClient.AccountID,
			Resource:  "association/" + aws.ToString(output.AssociationDescription.AssociationId),
		}.String()

		SetFrameworkFromString(&plan.Arn, amazonResourceName, false)
		SetFrameworkFromtStringPointer(&plan.AssociationId, output.AssociationDescription.AssociationId)
		SetFrameworkFromtStringPointer(&plan.AssociationVersion, output.AssociationDescription.AssociationVersion)
		SetFrameworkFromtStringPointer(&plan.DocumentVersion, output.AssociationDescription.DocumentVersion)
		plan.Parameters = parametersOut(output.AssociationDescription.Parameters)
		plan.Targets = targetsOut(ctx, output.AssociationDescription.Targets)
		plan.TagsAll = state.TagsAll
	}

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (a *AWSSSMAssociationResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data AWSSSMAssociationResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	_, err := a.Meta.AWSClient.SSMClient.DeleteAssociation(ctx, &ssm.DeleteAssociationInput{
		AssociationId: data.AssociationId.ValueStringPointer(),
	})
	if err != nil {
		response.Diagnostics.AddError("Error deleting SSM association", err.Error())
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (a *AWSSSMAssociationResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}

func FindAssociationByID(ctx context.Context, conn *ssm.Client, id string) (*awstypes.AssociationDescription, error) {
	input := &ssm.DescribeAssociationInput{
		AssociationId: aws.String(id),
	}

	output, err := conn.DescribeAssociation(ctx, input)

	if err != nil {
		return nil, err
	}

	if output == nil || output.AssociationDescription == nil || output.AssociationDescription.Overview == nil {
		return nil, errors.New("association does not exist")
	}

	return output.AssociationDescription, nil
}

func findAssociationTagsByID(ctx context.Context, conn *ssm.Client, id string) ([]awstypes.Tag, error) {
	input := &ssm.ListTagsForResourceInput{
		ResourceId:   aws.String(id),
		ResourceType: awstypes.ResourceTypeForTaggingAssociation,
	}

	output, err := conn.ListTagsForResource(ctx, input)

	if err != nil {
		return nil, err
	}

	return output.TagList, nil
}

func waitAssociationCreated(ctx context.Context, conn *ssm.Client, id string, timeout time.Duration, diag diag.Diagnostics) (*awstypes.AssociationDescription, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{string(awstypes.AssociationStatusNamePending)},
		Target:  []string{string(awstypes.AssociationStatusNameSuccess)},
		Refresh: statusAssociation(ctx, conn, id),
		Timeout: timeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*awstypes.AssociationDescription); ok {
		if status := awstypes.AssociationStatusName(aws.ToString(output.Overview.Status)); status == awstypes.AssociationStatusNameFailed {
			diag.AddError("Association error", aws.ToString(output.Overview.DetailedStatus))
		}

		return output, err
	}

	return nil, err
}

func statusAssociation(ctx context.Context, conn *ssm.Client, id string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := FindAssociationByID(ctx, conn, id)

		if err != nil {
			return nil, "", err
		}

		// Use the Overview.Status field instead of the root-level Status as DescribeAssociation
		// does not appear to return the root-level Status in the API response at this time.
		return output, aws.ToString(output.Overview.Status), nil
	}
}

func isAutoSSMTarget(targets []awstypes.Target) bool {
	if len(targets) == 1 {
		if aws.ToString(targets[0].Key) == "aws:NoOpAutomationTag" {
			if len(targets[0].Values) == 1 {
				if targets[0].Values[0] == "AWS-NoOpAutomationTarget-Value" {
					return true
				}
			}
		}
	}

	return false
}
