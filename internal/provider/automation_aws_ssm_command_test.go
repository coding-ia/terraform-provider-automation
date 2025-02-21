package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"testing"
)

func TestAccSSMCommand_basic(t *testing.T) {
	ctx := context.Background()
	rName := acctest.RandomWithPrefix("tf-acc-test")
	//resourceName := "automation_aws_ssm_command.test"

	resource.ParallelTest(t, resource.TestCase{
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
				Config: testAccCommandConfig_basic(rName),
				Check:  resource.ComposeTestCheckFunc(
				//testAccCheckAssociationExists(ctx, resourceName),
				//resource.TestCheckResourceAttr(resourceName, "apply_only_at_cron_interval", "false"),
				//resource.TestCheckResourceAttr(resourceName, "output_location.#", "0"),
				//resource.TestCheckResourceAttr(resourceName, "targets.#", "1"),
				//resource.TestCheckResourceAttr(resourceName, "targets.0.key", "InstanceIds"),
				//resource.TestCheckResourceAttr(resourceName, "targets.0.values.#", "1"),
				//resource.TestCheckResourceAttrPair(resourceName, "targets.0.values.0", "aws_instance.test", "id"),
				//resource.TestCheckResourceAttr(resourceName, "parameters.%", "0"),
				//resource.TestCheckResourceAttr(resourceName, "document_version", "$DEFAULT"),
				//resource.TestCheckResourceAttr(resourceName, "name", rName),
				),
			},
		},
	})
}

func testAccCommandConfig_basic(rName string) string {
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
    values = ["al2023-ami-*-kernel-6.1-x86_64"]
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

resource "aws_iam_role_policy_attachment" "ssm_policy_attachment" {
  role       = aws_iam_role.test.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2RoleforSSM"
}

resource "aws_iam_instance_profile" "test" {
  name = %[1]q
  role = aws_iam_role.test.name
}

resource "aws_instance" "test" {
  ami                     = data.aws_ami.amzn.image_id
  iam_instance_profile    = aws_iam_instance_profile.test.name
  instance_type           = "t2.micro"
  vpc_security_group_ids  = [aws_security_group.test.id]
  subnet_id               = data.aws_subnet.default.id

  tags = {
    Name = %[1]q
  }
}

resource "aws_ssm_document" "test" {
  name          = %[1]q
  document_type = "Command"

  content = <<DOC
{
  "schemaVersion": "1.2",
  "description": "Check ip configuration of a Linux instance.",
  "parameters": {},
  "runtimeConfig": {
    "aws:runShellScript": {
      "properties": [
        {
          "id": "0.aws:runShellScript",
          "runCommand": [
            "ifconfig"
          ]
        }
      ]
    }
  }
}
DOC

}

resource "automation_aws_ssm_command" "test" {
  create_command {
    document_name = aws_ssm_document.test.name
  }

  delete_command {
    document_name = aws_ssm_document.test.name
  }

  instance_ids                     = [aws_instance.test.id]
  wait_for_success_timeout_seconds = 3600
}
`, rName)
}
