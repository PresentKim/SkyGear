package sgitem

import (
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/presentkim/skygear/common/i18n"
)

// ConfigCopier is the item.
type ConfigCopier struct{}

func (ConfigCopier) EncodeItem() (string, int16) { return "skygear:config_copier", 0 }

func (ConfigCopier) Name() string { return i18n.T(i18n.Default, "copier.name") }

func (ConfigCopier) Category() category.Category { return category.Items() }

func (ConfigCopier) MaxCount() int { return 1 }
