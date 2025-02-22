package provider

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"testing"
)

func TestAccSSMStartAutomation_withParameters(t *testing.T) {
	ctx := context.Background()
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "automation_aws_ssm_start_automation.test"

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
				Config: testAccStartAutomationConfig_basicParameters(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "parameters.Directory.0", "myWorkSpace"),
				),
			},
			{
				Config: testAccStartAutomationConfig_basicParametersUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "parameters.Directory.0", "myWorkSpaceUpdated"),
				),
			},
		},
	})
}

func TestAccSSMStartAutomation_updateDefault(t *testing.T) {
	ctx := context.Background()
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "automation_aws_ssm_start_automation.test"

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
				Config: testAccStartAutomationConfig_basicDefault(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "document_version", "1"),
				),
			},
			{
				RefreshState: true,
			},
			{
				Config: testAccStartAutomationConfig_basicDefaultUpdated(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "document_version", "1"),
				),
			},
		},
	})
}

func TestAccSSMStartAutomation_stopOnDelete(t *testing.T) {
	ctx := context.Background()
	rName := acctest.RandomWithPrefix("tf-acc-test")
	resourceName := "automation_aws_ssm_start_automation.test"

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
				Config: testAccStartAutomationConfig_basicLongRunning(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckStartAutomationExists(ctx, resourceName),
				),
			},
			{
				Config:  testAccStartAutomationConfig_basicLongRunning(rName),
				Destroy: true,
			},
		},
	})
}

func testAccStartAutomationConfig_basicParameters(rName string) string {
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

resource "automation_aws_ssm_start_automation" "test" {
  document_name = aws_ssm_document.test.name

  parameters = {
    Directory = [ "myWorkSpace" ]
  }
}
`, rName)
}

func testAccStartAutomationConfig_basicParametersUpdated(rName string) string {
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

resource "automation_aws_ssm_start_automation" "test" {
  document_name = aws_ssm_document.test.name

  parameters = {
    Directory = [ "myWorkSpaceUpdated" ]
  }
}
`, rName)
}

func testAccStartAutomationConfig_basicDefault(rName string) string {
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

resource "automation_aws_ssm_start_automation" "test" {
  document_name = aws_ssm_document.test.name
}
`, rName)
}

func testAccStartAutomationConfig_basicDefaultUpdated(rName string) string {
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

resource "automation_aws_ssm_start_automation" "test" {
  document_name    = aws_ssm_document.test.name
  document_version = "1"
}
`, rName)
}

func testAccStartAutomationConfig_basicLongRunning(rName string) string {
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

resource "automation_aws_ssm_start_automation" "test" {
  document_name = aws_ssm_document.test.name
}
`, rName)
}

func testAccCheckStartAutomationExists(ctx context.Context, n string) resource.TestCheckFunc {
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
