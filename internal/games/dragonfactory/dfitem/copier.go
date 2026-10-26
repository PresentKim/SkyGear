package dfitem

import (
	"github.com/df-mc/dragonfly/server/item/category"
	"github.com/presentkim/dragonfactory/common/i18n"
)

// ConfigCopier is the item.
type ConfigCopier struct{}

func (ConfigCopier) EncodeItem() (string, int16) { return "dragonfactory:config_copier", 0 }

func (ConfigCopier) Name() string { return i18n.T(i18n.Default, "copier.name") }

func (ConfigCopier) Category() category.Category { return category.Items() }

func (ConfigCopier) MaxCount() int { return 1 }
