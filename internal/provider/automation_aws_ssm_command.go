package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"slices"
	"time"
)

var _ resource.Resource = &AWSSSMCommandResource{}
var _ resource.ResourceWithConfigure = &AWSSSMCommandResource{}

type AWSSSMCommandResource struct {
	Meta Meta
}

type AWSSSMCommandResourceModel struct {
	CreateCommand                CreateCommandModel    `tfsdk:"create_command"`
	DeleteCommand                []DeleteCommandModel  `tfsdk:"delete_command"`
	InstanceIds                  types.List            `tfsdk:"instance_ids"`
	MaxConcurrency               types.String          `tfsdk:"max_concurrency"`
	MaxErrors                    types.String          `tfsdk:"max_errors"`
	OutputLocation               []OutputLocationModel `tfsdk:"output_location"`
	WaitForSuccessTimeoutSeconds types.Int32           `tfsdk:"wait_for_success_timeout_seconds"`
}

type CreateCommandModel struct {
	CommandID       types.String `tfsdk:"command_id"`
	Comment         types.String `tfsdk:"comment"`
	DocumentName    types.String `tfsdk:"document_name"`
	DocumentVersion types.String `tfsdk:"document_version"`
	Parameters      types.Map    `tfsdk:"parameters"`
}

type DeleteCommandModel struct {
	Comment         types.String `tfsdk:"comment"`
	DocumentName    types.String `tfsdk:"document_name"`
	DocumentVersion types.String `tfsdk:"document_version"`
	Parameters      types.Map    `tfsdk:"parameters"`
}

func newAWSSSMCommandResource() resource.Resource {
	return &AWSSSMCommandResource{}
}

func (c *AWSSSMCommandResource) Configure(_ context.Context, request resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	c.Meta = request.ProviderData.(Meta)
}

func (c *AWSSSMCommandResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_aws_ssm_command"
}

func (c *AWSSSMCommandResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Sends SSM Command instance.",
		Attributes: map[string]schema.Attribute{
			"instance_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeBetween(0, 50),
				},
			},
			"max_concurrency": schema.StringAttribute{
				Description: "The maximum number of targets allowed to run the association at the same time.  You can specify a number, for example 10, or a percentage of the target set, for example 10%.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexache.MustCompile(`^([1-9][0-9]*|[1-9][0-9]%|[1-9]%|100%)$`), "must be a valid number (e.g. 10) or percentage including the percent sign (e.g. 10%)"),
				},
			},
			"max_errors": schema.StringAttribute{
				Description: "The number of errors that are allowed before the system stops sending requests to run the association on additional targets.  You can specify either an absolute number of errors, for example 10, or a percentage of the target set, for example 10%.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexache.MustCompile(`^([1-9][0-9]*|[0]|[1-9][0-9]%|[0-9]%|100%)$`), "must be a valid number (e.g. 10) or percentage including the percent sign (e.g. 10%)"),
				},
			},
			"wait_for_success_timeout_seconds": schema.Int32Attribute{
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"create_command": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"command_id": schema.StringAttribute{
						Computed: true,
					},
					"comment": schema.StringAttribute{
						Description: "User-specified information about the command, such as a brief description of what the command should do.",
						Optional:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtMost(100),
						},
					},
					"document_name": schema.StringAttribute{
						Description: "The name of the SSM Command document or Automation runbook.",
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
						Default: stringdefault.StaticString("$DEFAULT"),
					},
					"parameters": schema.MapAttribute{
						Description: "The parameters for the runtime configuration of the document.",
						Optional:    true,
						Computed:    true,
						ElementType: types.ListType{ElemType: types.StringType},
					},
				},
			},
			"delete_command": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"comment": schema.StringAttribute{
							Description: "User-specified information about the command, such as a brief description of what the command should do.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtMost(100),
							},
						},
						"document_name": schema.StringAttribute{
							Description: "The name of the SSM Command document or Automation runbook.",
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
							Default: stringdefault.StaticString("$DEFAULT"),
						},
						"parameters": schema.MapAttribute{
							Description: "The parameters for the runtime configuration of the document.",
							Optional:    true,
							ElementType: types.ListType{ElemType: types.StringType},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"output_location": schema.ListNestedBlock{
				Description: "An Amazon Simple Storage Service (Amazon S3) bucket where you want to store the output details of the request.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"s3_bucket_name": schema.StringAttribute{
							Description: "The name of the S3 bucket.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(3, 63),
							},
						},
						"s3_key_prefix": schema.StringAttribute{
							Description: "The S3 bucket subfolder.",
							Optional:    true,
							Validators: []validator.String{
								stringvalidator.LengthBetween(0, 500),
							},
						},
						"s3_region": schema.StringAttribute{
							Description: "The AWS Region of the S3 bucket.",
							Optional:    true,
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

func (c *AWSSSMCommandResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data AWSSSMCommandResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	input := &ssm.SendCommandInput{
		DocumentName: data.CreateCommand.DocumentName.ValueStringPointer(),
	}

	if !data.CreateCommand.Comment.IsNull() {
		input.Comment = data.CreateCommand.Comment.ValueStringPointer()
	}

	if !data.CreateCommand.DocumentVersion.IsNull() {
		input.DocumentVersion = data.CreateCommand.DocumentVersion.ValueStringPointer()
	}

	ssmClient := c.Meta.AWSClient.SSMClient

	if !data.InstanceIds.IsNull() {
		instanceIds := make([]string, 0, len(data.InstanceIds.Elements()))
		diags := data.InstanceIds.ElementsAs(ctx, &instanceIds, false)

		if diags != nil {
			response.Diagnostics.Append(diags...)
			return
		}

		input.InstanceIds = instanceIds

		err := waitForInstancesOnline(ctx, ssmClient, input.InstanceIds)
		if err != nil {
			response.Diagnostics.AddError("Error waiting for instances", err.Error())
			return
		}
	}

	output, err := ssmClient.SendCommand(ctx, input)
	if err != nil {
		response.Diagnostics.AddError("Error sending SSM command", err.Error())
		return
	}

	data.CreateCommand.CommandID = types.StringPointerValue(output.Command.CommandId)
	data.CreateCommand.DocumentVersion = types.StringPointerValue(output.Command.DocumentVersion)

	if data.CreateCommand.Parameters.IsNull() || data.CreateCommand.Parameters.IsUnknown() {
		data.CreateCommand.Parameters = parametersOut(output.Command.Parameters)
	}

	if !data.WaitForSuccessTimeoutSeconds.IsNull() {
		timeout := time.Duration(data.WaitForSuccessTimeoutSeconds.ValueInt32()) * time.Second
		if _, err = waitCommandComplete(ctx, ssmClient, output.Command.CommandId, timeout, response.Diagnostics); err != nil {
			response.Diagnostics.AddError("Error running SSM command", fmt.Sprintf("waiting for SSM command (%s) complete: %s", aws.ToString(output.Command.CommandId), err.Error()))
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (c *AWSSSMCommandResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data AWSSSMCommandResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (c *AWSSSMCommandResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan, state AWSSSMCommandResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (c *AWSSSMCommandResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data AWSSSMCommandResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	if data.DeleteCommand != nil {

	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func FindCommandByID(ctx context.Context, conn *ssm.Client, id *string) (*awstypes.Command, error) {
	input := &ssm.ListCommandsInput{
		CommandId: id,
	}

	output, err := conn.ListCommands(ctx, input)

	if err != nil {
		return nil, err
	}

	if output == nil || output.Commands == nil {
		return nil, errors.New("association does not exist")
	}

	return &output.Commands[0], nil
}

func checkInstancesOnline(ctx context.Context, client *ssm.Client, instanceIDs []string) *retry.RetryError {
	iif := []awstypes.InstanceInformationFilter{
		{
			Key:      awstypes.InstanceInformationFilterKeyInstanceIds,
			ValueSet: instanceIDs,
		},
	}

	input := &ssm.DescribeInstanceInformationInput{
		InstanceInformationFilterList: iif,
		MaxResults:                    aws.Int32(50),
	}
	output, err := client.DescribeInstanceInformation(ctx, input)
	if err != nil {
		return retry.NonRetryableError(err)
	}

	if len(instanceIDs) != len(output.InstanceInformationList) {
		return retry.RetryableError(fmt.Errorf("%d of %d instances are available", len(instanceIDs), len(output.InstanceInformationList)))
	}

	for _, instance := range output.InstanceInformationList {
		id := aws.ToString(instance.InstanceId)
		if !slices.Contains(instanceIDs, id) {
			return retry.RetryableError(fmt.Errorf("instance %s is not online yet", id))
		}
	}

	return nil
}

func waitForInstancesOnline(ctx context.Context, conn *ssm.Client, instanceIDs []string) error {
	timeout := 3600 * time.Second
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		return checkInstancesOnline(ctx, conn, instanceIDs)
	})

	return err
}

func waitCommandComplete(ctx context.Context, conn *ssm.Client, commandId *string, timeout time.Duration, diag diag.Diagnostics) (*awstypes.Command, error) {
	stateConf := &retry.StateChangeConf{
		Pending: []string{
			string(awstypes.CommandStatusPending),
			string(awstypes.CommandStatusInProgress),
		},
		Target:  []string{string(awstypes.CommandStatusSuccess)},
		Refresh: statusCommand(ctx, conn, commandId),
		Timeout: timeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*awstypes.Command); ok {
		if output.Status == awstypes.CommandStatusFailed {
			diag.AddError("Command error", "Command status is failed.")
		}

		return output, err
	}

	return nil, err
}

func statusCommand(ctx context.Context, conn *ssm.Client, commandId *string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		command, err := FindCommandByID(ctx, conn, commandId)

		return command, string(command.Status), err
	}
}
