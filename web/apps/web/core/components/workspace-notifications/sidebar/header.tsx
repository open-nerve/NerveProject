/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import { InboxOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { Breadcrumbs, Header } from "@nerve/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
// local imports
import { NotificationHeaderOptions } from "./header-options";

/**
 * The head of the notifications page's sidebar (P4 spec 2.6): its crumb, the inbox; then mark all read, refresh, the
 * type filter and the view menu (NotificationHeaderOptions).
 */
export function NotificationSidebarHeader() {
  const { t } = useTranslation();
  return (
    <Header className="my-auto bg-surface-1">
      <Header.LeftItem>
        <Breadcrumbs>
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label={t("notification.label")}
                icon={<InboxOutline className="h-4 w-4 text-primary" />}
                disableTooltip
              />
            }
          />
        </Breadcrumbs>
      </Header.LeftItem>
      <Header.RightItem>
        <NotificationHeaderOptions />
      </Header.RightItem>
    </Header>
  );
}
