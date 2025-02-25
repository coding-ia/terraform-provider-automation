package provider

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"testing"
)

func TestAccSSMStartAutomationExecution_withParameters(t *testing.T) {
	ctx := context.Background()
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "automation_aws_ssm_start_automation_execution.test"

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "5.87.0",
			},
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAssociationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccStartAutomationExecutionConfig_basicParameters(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExecutionExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "parameters.Directory.0", "myWorkSpace"),
				),
			},
			{
				Config: testAccStartAutomationExecutionConfig_basicParametersUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExecutionExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "parameters.Directory.0", "myWorkSpaceUpdated"),
				),
			},
		},
	})
}

func TestAccSSMStartAutomationExecution_updateDefault(t *testing.T) {
	ctx := context.Background()
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "automation_aws_ssm_start_automation_execution.test"

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "5.87.0",
			},
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAssociationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccStartAutomationExecutionConfig_basicDefault(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExecutionExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "document_version", "1"),
				),
			},
			{
				RefreshState: true,
			},
			{
				Config: testAccStartAutomationExecutionConfig_basicDefaultUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExecutionExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "document_version", "1"),
				),
			},
		},
	})
}

func TestAccSSMStartAutomationExecution_stopOnDelete(t *testing.T) {
	ctx := context.Background()
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "automation_aws_ssm_start_automation_execution.test"

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "5.87.0",
			},
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAssociationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccStartAutomationExecutionConfig_basicLongRunning(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExecutionExists(ctx, resourceName),
				),
			},
			{
				RefreshState: true,
			},
			{
				Config:  testAccStartAutomationExecutionConfig_basicLongRunning(rName),
				Destroy: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExecutionStatus(ctx, resourceName, string(awstypes.AutomationExecutionStatusCancelled)),
				),
			},
		},
	})
}

func TestAccSSMStartAutomationExecution_WaitForSuccessTimeout(t *testing.T) {
	ctx := context.Background()
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "automation_aws_ssm_start_automation_execution.test"

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "5.87.0",
			},
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAssociationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccStartAutomationExecutionConfig_basicLongRunningWithWaitForSuccess(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExecutionExists(ctx, resourceName),
				),
			},
		},
	})
}

func TestAccSSMStartAutomationExecution_Basic(t *testing.T) {
	ctx := context.Background()
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "automation_aws_ssm_start_automation_execution.test"

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": {
				Source:            "hashicorp/aws",
				VersionConstraint: "5.87.0",
			},
		},
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckAssociationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccStartAutomationExecutionConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExecutionExists(ctx, resourceName),
				),
			},
		},
	})
}

func testAccStartAutomationExecutionConfig_basicParameters(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "test" {
  name          = "%[1]s-2"
  document_type = "Automation"

  content = <<-DOC
{
  "schemaVersion": "0.3",
  "parameters": {
    "Directory": {
      "type": "String",
      "default": "",
      "description": "(Optional) The path to the working directory on your instance."
    }
  },
  "mainSteps": [
    {
      "name": "Sleep",
      "action": "aws:sleep",
      "isEnd": true,
      "inputs": {
        "Duration": "PT10S"
      }
    }
  ]
}
  DOC

}

resource "automation_aws_ssm_start_automation_execution" "test" {
  document_name = aws_ssm_document.test.name

  parameters = {
    Directory = [ "myWorkSpace" ]
  }
}
`, rName)
}

func testAccStartAutomationExecutionConfig_basicParametersUpdated(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "test" {
  name          = "%[1]s-2"
  document_type = "Automation"

  content = <<-DOC
{
  "schemaVersion": "0.3",
  "parameters": {
    "Directory": {
      "type": "String",
      "default": "",
      "description": "(Optional) The path to the working directory on your instance."
    }
  },
  "mainSteps": [
    {
      "name": "Sleep",
      "action": "aws:sleep",
      "isEnd": true,
      "inputs": {
        "Duration": "PT10S"
      }
    }
  ]
}
  DOC

}

resource "automation_aws_ssm_start_automation_execution" "test" {
  document_name = aws_ssm_document.test.name

  parameters = {
    Directory = [ "myWorkSpaceUpdated" ]
  }
}
`, rName)
}

func testAccStartAutomationExecutionConfig_basicDefault(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "test" {
  name          = "%[1]s"
  document_type = "Automation"

  content = <<-DOC
{
  "schemaVersion": "0.3",
  "mainSteps": [
    {
      "name": "Sleep",
      "action": "aws:sleep",
      "isEnd": true,
      "inputs": {
        "Duration": "PT10S"
      }
    }
  ]
}
  DOC

}

resource "automation_aws_ssm_start_automation_execution" "test" {
  document_name = aws_ssm_document.test.name
}
`, rName)
}

func testAccStartAutomationExecutionConfig_basicDefaultUpdated(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "test" {
  name          = "%[1]s"
  document_type = "Automation"

  content = <<-DOC
{
  "schemaVersion": "0.3",
  "mainSteps": [
    {
      "name": "Sleep",
      "action": "aws:sleep",
      "isEnd": true,
      "inputs": {
        "Duration": "PT10S"
      }
    }
  ]
}
  DOC

}

resource "automation_aws_ssm_start_automation_execution" "test" {
  document_name    = aws_ssm_document.test.name
  document_version = "1"
}
`, rName)
}

func testAccStartAutomationExecutionConfig_basicLongRunning(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "test" {
  name          = "%[1]s"
  document_type = "Automation"

  content = <<-DOC
{
  "schemaVersion": "0.3",
  "mainSteps": [
    {
      "name": "Sleep",
      "action": "aws:sleep",
      "isEnd": true,
      "inputs": {
        "Duration": "PT5M"
      }
    }
  ]
}
  DOC

}

resource "automation_aws_ssm_start_automation_execution" "test" {
  document_name = aws_ssm_document.test.name
}
`, rName)
}

func testAccStartAutomationExecutionConfig_basicLongRunningWithWaitForSuccess(rName string) string {
	return fmt.Sprintf(`
resource "aws_ssm_document" "test" {
  name          = "%[1]s"
  document_type = "Automation"

  content = <<-DOC
{
  "schemaVersion": "0.3",
  "mainSteps": [
    {
      "name": "Sleep",
      "action": "aws:sleep",
      "isEnd": true,
      "inputs": {
        "Duration": "PT1M"
      }
    }
  ]
}
  DOC

}

resource "automation_aws_ssm_start_automation_execution" "test" {
  document_name                    = aws_ssm_document.test.name
  wait_for_success_timeout_seconds = 90
}
`, rName)
}

func testAccStartAutomationExecutionConfig_basic(rName string) string {
	return fmt.Sprintf(`
data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

data "aws_ami" "amzn" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["amzn2-ami-hvm-*-x86_64-gp2"]
  }
}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnet" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
  filter {
    name   = "availabilityZone"
    values = [data.aws_availability_zones.available.names[0]]  # Replace with your desired availability zone
  }
}

resource "aws_security_group" "test" {
  name        = %[1]q
  description = "foo"
  vpc_id      = data.aws_vpc.default.id

  egress = [
    {
      cidr_blocks = [
        "0.0.0.0/0",
      ]
      description = ""
      from_port   = 0
      ipv6_cidr_blocks = [
        "::/0",
      ]
      prefix_list_ids = []
      protocol        = "-1"
      security_groups = []
      self            = false
      to_port         = 0
    },
  ]
  ingress {
    protocol    = "icmp"
    from_port   = -1
    to_port     = -1
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_iam_role" "test" {
  name = %[1]q
  
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
        Action = "sts:AssumeRole"
      }
    ]
  })
}
resource "aws_iam_role_policy_attachment" "test" {
  role       = aws_iam_role.test.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2RoleforSSM"
}
resource "aws_iam_instance_profile" "test" {
  name = %[1]q
  role = aws_iam_role.test.name
}

resource "aws_instance" "test" {
  ami                    = data.aws_ami.amzn.image_id
  availability_zone      = data.aws_availability_zones.available.names[0]
  iam_instance_profile   = aws_iam_instance_profile.test.name
  instance_type          = "t2.micro"
  vpc_security_group_ids = [aws_security_group.test.id]
  subnet_id              = data.aws_subnet.default.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_ssm_document" "test" {
  name          = "%[1]s"
  document_type = "Automation"

  content = <<-DOC
{
  "schemaVersion": "0.3",
  "description": "Unit test automation.",
  "assumeRole": "{{AutomationAssumeRole}}",
  "parameters": {
    "AutomationAssumeRole": {
      "type": "String",
      "default": ""
    },
    "InstanceId": {
      "type": "String"
    }
  },
  "mainSteps": [
    {
      "name": "RunCommandOnInstances",
      "action": "aws:runCommand",
      "isEnd": true,
      "inputs": {
        "DocumentName": "AWS-RunShellScript",
        "Parameters": {
          "commands": [
            "ifconfig"
          ]
        },
        "InstanceIds": [
          "{{ InstanceId }}"
        ]
      }
    }
  ]
}
  DOC

}

resource "automation_aws_ssm_start_automation_execution" "test" {
  document_name = aws_ssm_document.test.name

  parameters = {
    InstanceId           = [ aws_instance.test.id ]
  }
}
`, rName)
}

func testAccCheckStartAutomationExecutionExists(ctx context.Context, n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		conn := getProviderMeta(ctx).AWSClient.SSMClient

		automationId := rs.Primary.Attributes["automation_id"]
		_, err := FindAutomationExecutionById(ctx, conn, aws.String(automationId))

		return err
	}
}

func testAccCheckStartAutomationExecutionStatus(ctx context.Context, n string, status string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		conn := getProviderMeta(ctx).AWSClient.SSMClient

		automationId := rs.Primary.Attributes["automation_id"]
		ae, err := FindAutomationExecutionById(ctx, conn, aws.String(automationId))

		if err != nil {
			return err
		}

		if string(ae.AutomationExecutionStatus) != status {
			return fmt.Errorf("automation status does not match %s", status)
		}

		return nil
	}
}
