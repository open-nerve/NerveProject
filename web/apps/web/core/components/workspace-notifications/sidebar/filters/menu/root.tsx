/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { FilterOutline } from "@makeplane/propel/icons";
// nerve imports
import type { ENotificationFilterType } from "@nerve/constants";
import { FILTER_TYPE_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { PopoverMenu } from "@nerve/ui";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { NotificationFilterOptionItem } from "./menu-option-item";
import { IconButton } from "@nerve/propel/icon-button";

export const NotificationFilter = observer(function NotificationFilter() {
  // hooks
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();

  const translatedFilterTypeOptions = FILTER_TYPE_OPTIONS.map((filter) => ({
    ...filter,
    label: t(filter.i18n_label),
  }));

  return (
    <PopoverMenu
      data={translatedFilterTypeOptions}
      button={
        <Tooltip label={t("notification.options.filters")} side="bottom" disabled={isMobile}>
          <IconButton size="base" variant="ghost" icon={FilterOutline} />
        </Tooltip>
      }
      keyExtractor={(item: { label: string; value: ENotificationFilterType }) => item.value}
      render={(item) => <NotificationFilterOptionItem {...item} />}
    />
  );
});
