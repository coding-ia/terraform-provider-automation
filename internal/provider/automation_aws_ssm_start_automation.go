package provider

import (
	"context"
	"fmt"
	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/coding-ia/terraform-provider-automation/internal/framework/errs"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"time"
)

var _ resource.Resource = &AWSSSMStartAutomationResource{}
var _ resource.ResourceWithConfigure = &AWSSSMStartAutomationResource{}

type AWSSSMStartAutomationResource struct {
	Meta Meta
}

type AWSSSMStartAutomationResourceModel struct {
	AutomationId                 types.String `tfsdk:"automation_id"`
	ClientToken                  types.String `tfsdk:"client_token"`
	DocumentName                 types.String `tfsdk:"document_name"`
	DocumentVersion              types.String `tfsdk:"document_version"`
	MaxConcurrency               types.String `tfsdk:"max_concurrency"`
	MaxErrors                    types.String `tfsdk:"max_errors"`
	Mode                         types.String `tfsdk:"mode"`
	Parameters                   types.Map    `tfsdk:"parameters"`
	Tags                         types.Map    `tfsdk:"tags"`
	TargetParameterName          types.String `tfsdk:"target_parameter_name"`
	Targets                      types.List   `tfsdk:"targets"`
	WaitForSuccessTimeoutSeconds types.Int32  `tfsdk:"wait_for_success_timeout_seconds"`
}

func newAWSSSMStartAutomationResource() resource.Resource {
	return &AWSSSMStartAutomationResource{}
}

func (a *AWSSSMStartAutomationResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	a.Meta = request.ProviderData.(Meta)
}

func (a *AWSSSMStartAutomationResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_aws_ssm_start_automation"
}

func (a *AWSSSMStartAutomationResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Start automation of an SSM Document to an instance or EC2 tag.",
		Attributes: map[string]schema.Attribute{
			"automation_id": schema.StringAttribute{
				Description: "The ID of the automation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_token": schema.StringAttribute{
				Description: "Generated idempotency token.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"document_name": schema.StringAttribute{
				Description: "The name of the SSM automation document.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"document_version": schema.StringAttribute{
				Description: "The document version you want to associate with the targets.  Can be a specific version or the default version.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexache.MustCompile(`^([$]LATEST|[$]DEFAULT|^[1-9][0-9]*$)$`), ""),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"max_concurrency": schema.StringAttribute{
				Description: "The maximum number of targets allowed to run the association at the same time.  You can specify a number, for example 10, or a percentage of the target set, for example 10%.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexache.MustCompile(`^([1-9][0-9]*|[1-9][0-9]%|[1-9]%|100%)$`), "must be a valid number (e.g. 10) or percentage including the percent sign (e.g. 10%)"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"max_errors": schema.StringAttribute{
				Description: "The number of errors that are allowed before the system stops sending requests to run the association on additional targets.  You can specify either an absolute number of errors, for example 10, or a percentage of the target set, for example 10%.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexache.MustCompile(`^([1-9][0-9]*|[0]|[1-9][0-9]%|[0-9]%|100%)$`), "must be a valid number (e.g. 10) or percentage including the percent sign (e.g. 10%)"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mode": schema.StringAttribute{
				Description: "The execution mode of the automation..",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive(
						[]string{
							"Auto",
							"Interactive",
						}...,
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parameters": schema.MapAttribute{
				Description: "The parameters for the runtime configuration of the document.",
				Optional:    true,
				ElementType: types.ListType{ElemType: types.StringType},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"tags": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"target_parameter_name": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 50),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"targets": schema.ListAttribute{
				Description: "The targets for the association.  You can target managed nodes by using tags, AWS resource groups, all managed nodes in an AWS account, or individual managed node IDs.",
				Optional:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"key":    types.StringType,
						"values": types.ListType{ElemType: types.StringType},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(5),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"wait_for_success_timeout_seconds": schema.Int32Attribute{
				Optional: true,
			},
		},
	}

}

func (a *AWSSSMStartAutomationResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data AWSSSMStartAutomationResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	ssmClient := a.Meta.AWSClient.SSMClient
	err := StartAutomationExecution(ctx, ssmClient, &data)
	if err != nil {
		response.Diagnostics.AddError("Error starting automation execution", err.Error())
		return
	}

	if !data.WaitForSuccessTimeoutSeconds.IsNull() &&
		!data.WaitForSuccessTimeoutSeconds.IsUnknown() {
		timeout := time.Duration(data.WaitForSuccessTimeoutSeconds.ValueInt32()) * time.Second
		if _, err := waitStartAutomation(ctx, ssmClient, data.AutomationId.ValueStringPointer(), timeout, response.Diagnostics); err != nil {
			response.Diagnostics.AddError("Error creating SSM association", fmt.Sprintf("waiting for SSM Association (%s) create: %s", data.AutomationId.String(), err.Error()))
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (a *AWSSSMStartAutomationResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data AWSSSMStartAutomationResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (a *AWSSSMStartAutomationResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state AWSSSMStartAutomationResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (a *AWSSSMStartAutomationResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data AWSSSMStartAutomationResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	ssmClient := a.Meta.AWSClient.SSMClient
	ae, err := FindAutomationExecutionById(ctx, ssmClient, data.AutomationId.ValueStringPointer())
	if err != nil {

	}

	if ae != nil {
		if ae.AutomationExecutionStatus == awstypes.AutomationExecutionStatusInprogress ||
			ae.AutomationExecutionStatus == awstypes.AutomationExecutionStatusPending ||
			ae.AutomationExecutionStatus == awstypes.AutomationExecutionStatusPendingApproval {
			_ = StopAutomationExecution(ctx, ssmClient, ae.AutomationExecutionId)
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func StartAutomationExecution(ctx context.Context, conn *ssm.Client, data *AWSSSMStartAutomationResourceModel) error {
	input := &ssm.StartAutomationExecutionInput{
		DocumentName: data.DocumentName.ValueStringPointer(),
	}

	if !data.ClientToken.IsNull() {
		input.ClientToken = data.ClientToken.ValueStringPointer()
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

	if !data.Mode.IsNull() {
		input.Mode = awstypes.ExecutionMode(data.Mode.ValueString())
	}

	if !data.Parameters.IsNull() && !data.Parameters.IsUnknown() {
		input.Parameters = parametersIn(ctx, data.Parameters.Elements())
	}

	if !data.Tags.IsNull() && !data.Parameters.IsUnknown() {
		input.Tags = tagsIn(data.Tags.Elements())
	}

	if !data.TargetParameterName.IsNull() {
		input.TargetParameterName = data.TargetParameterName.ValueStringPointer()
	}

	if !data.Targets.IsNull() && !data.Targets.IsUnknown() {
		input.Targets = targetsIn(data.Targets)
	}

	output, err := conn.StartAutomationExecution(ctx, input)
	if err != nil {
		return err
	}

	ae, err := FindAutomationExecutionById(ctx, conn, output.AutomationExecutionId)
	if err != nil {
		return err
	}

	data.AutomationId = types.StringPointerValue(output.AutomationExecutionId)

	if ae != nil {
		data.DocumentVersion = types.StringPointerValue(ae.DocumentVersion)
	}

	return nil
}

func FindAutomationExecutionById(ctx context.Context, conn *ssm.Client, id *string) (*awstypes.AutomationExecution, error) {
	input := &ssm.GetAutomationExecutionInput{
		AutomationExecutionId: id,
	}

	output, err := conn.GetAutomationExecution(ctx, input)
	if err != nil {
		if errs.IsA[*awstypes.AutomationExecutionNotFoundException](err) {
			return nil, nil
		}

		return nil, err
	}

	return output.AutomationExecution, nil
}

func StopAutomationExecution(ctx context.Context, conn *ssm.Client, id *string) error {
	input := &ssm.StopAutomationExecutionInput{
		AutomationExecutionId: id,
	}

	_, err := conn.StopAutomationExecution(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

func waitStartAutomation(ctx context.Context, conn *ssm.Client, id *string, timeout time.Duration, diag diag.Diagnostics) (*awstypes.AutomationExecution, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(awstypes.AutomationExecutionStatusPending),
			string(awstypes.AutomationExecutionStatusInprogress),
		},
		Target: []string{
			string(awstypes.AutomationExecutionStatusCompletedWithSuccess),
			string(awstypes.AutomationExecutionStatusCompletedWithFailure),
		},
		Refresh: statusExecution(ctx, conn, id),
		Timeout: timeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*awstypes.AutomationExecution); ok {
		if status := string(output.AutomationExecutionStatus); status == string(awstypes.AutomationExecutionStatusFailed) {
			diag.AddError("Association error", string(output.AutomationExecutionStatus))
		}

		return output, err
	}

	return nil, err
}

func statusExecution(ctx context.Context, conn *ssm.Client, id *string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := FindAutomationExecutionById(ctx, conn, id)

		if err != nil {
			return nil, "", err
		}

		// Use the Overview.Status field instead of the root-level Status as DescribeAssociation
		// does not appear to return the root-level Status in the API response at this time.
		return output, string(output.AutomationExecutionStatus), nil
	}
}
