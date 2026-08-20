// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package resource_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
)

func TestAccResourceGroup_defaultTags(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_resource_group", "test")
	r := ResourceGroupResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			// provider defaults merge underneath resource-level tags
			Config: r.defaultTagsBasic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("tags.%").HasValue("3"),
				check.That(data.ResourceName).Key("tags.environment").HasValue("prod"),
				check.That(data.ResourceName).Key("tags.team").HasValue("platform"),
				check.That(data.ResourceName).Key("tags.cost_center").HasValue("msft"),
			),
		},
		{
			// a resource-level tag overrides the default with the same key
			Config: r.defaultTagsOverride(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("tags.environment").HasValue("dev"),
				check.That(data.ResourceName).Key("tags.team").HasValue("platform"),
			),
		},
		{
			// removing the defaults removes the tags they applied
			Config: r.defaultTagsRemoved(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("tags.%").HasValue("1"),
				check.That(data.ResourceName).Key("tags.cost_center").HasValue("msft"),
			),
		},
	})
}

func (r ResourceGroupResource) defaultTagsBasic(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}

  default_tags {
    tags = {
      environment = "prod"
      team        = "platform"
    }
  }
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"

  tags = {
    cost_center = "msft"
  }
}
`, data.RandomInteger, data.Locations.Primary)
}

func (r ResourceGroupResource) defaultTagsOverride(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}

  default_tags {
    tags = {
      environment = "prod"
      team        = "platform"
    }
  }
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"

  tags = {
    cost_center = "msft"
    environment = "dev"
  }
}
`, data.RandomInteger, data.Locations.Primary)
}

func (r ResourceGroupResource) defaultTagsRemoved(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-%d"
  location = "%s"

  tags = {
    cost_center = "msft"
  }
}
`, data.RandomInteger, data.Locations.Primary)
}
