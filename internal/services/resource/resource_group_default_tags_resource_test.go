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
			// changing the provider's default_tags value updates the merged tag in place
			Config: r.defaultTagsChanged(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("tags.environment").HasValue("staging"),
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
		{
			// establishes a resource whose tags are frozen by ignore_changes, with defaults merged in
			Config: r.defaultTagsIgnoreChanges(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("tags.%").HasValue("3"),
				check.That(data.ResourceName).Key("tags.cost_center").HasValue("msft"),
				check.That(data.ResourceName).Key("tags.environment").HasValue("prod"),
				check.That(data.ResourceName).Key("tags.team").HasValue("platform"),
			),
		},
		{
			// a resource-level tag edit stays frozen by ignore_changes while the provider defaults still apply
			Config: r.defaultTagsIgnoreChangesEdited(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("tags.cost_center").HasValue("msft"),
				check.That(data.ResourceName).Key("tags.environment").HasValue("prod"),
				check.That(data.ResourceName).Key("tags.team").HasValue("platform"),
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

func (r ResourceGroupResource) defaultTagsChanged(data acceptance.TestData) string {
	return fmt.Sprintf(`
provider "azurerm" {
  features {}

  default_tags {
    tags = {
      environment = "staging"
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

func (r ResourceGroupResource) defaultTagsIgnoreChanges(data acceptance.TestData) string {
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

  lifecycle {
    ignore_changes = [tags]
  }
}
`, data.RandomInteger, data.Locations.Primary)
}

func (r ResourceGroupResource) defaultTagsIgnoreChangesEdited(data acceptance.TestData) string {
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
    cost_center = "changed"
  }

  lifecycle {
    ignore_changes = [tags]
  }
}
`, data.RandomInteger, data.Locations.Primary)
}
