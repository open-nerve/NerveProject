/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { Tab, TabGroup, TabList, TabPanel, TabPanels } from "@headlessui/react";
// nerve imports
import { NOTIFICATION_TABS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { cn, getNumberCount } from "@nerve/utils";
// components
import { CountChip } from "@/components/common/count-chip";
// hooks
import { useUnreadNotificationCount } from "@/hooks/use-unread-notification-count";
// local imports
import { useNotificationPageState } from "../page-state";

type Props = {
  /** What the chosen tab's panel shows: the list of its notifications. */
  children: ReactNode;
};

/**
 * The notifications page's tabs, all and mentions (P4 spec 2.6, R5, R16): tabs as a screen reader reads them
 * (Headless Tab), the page's tab chosen. Each says its unread notifications, as the one fetch of the route's
 * workspace's counts gives them (useUnreadNotificationCount, which the top navigation shares): all for the all tab,
 * mentions for the mentions tab, nothing at 0, 99+ past 99 (getNumberCount); its name says them as well. Choosing one
 * sets the page's tab; its panel shows the children.
 */
export const NotificationTabs = observer(function NotificationTabs({ children }: Props) {
  const { t } = useTranslation();
  const { filters, setTab } = useNotificationPageState();
  const count = useUnreadNotificationCount();
  return (
    <TabGroup
      as="div"
      className="flex min-h-0 flex-1 flex-col"
      selectedIndex={Math.max(
        0,
        NOTIFICATION_TABS.findIndex((tab) => tab.value === filters.tab)
      )}
      onChange={(index: number) => setTab(NOTIFICATION_TABS[index]?.value ?? "all")}
    >
      <TabList className="flex h-10 flex-shrink-0 items-stretch border-b border-subtle px-3">
        {NOTIFICATION_TABS.map((tab) => {
          const label = t(tab.i18n_label);
          const unread = count?.[tab.value] ?? 0;
          const shown = unread > 0 ? getNumberCount(unread) : undefined;
          return (
            <Tab
              key={tab.value}
              aria-label={shown === undefined ? label : t("notification.a11y.unread_count", { label, count: shown })}
              className={({ selected }) =>
                cn(
                  "relative flex items-center gap-1 px-3 text-body-xs-medium transition-all focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong",
                  selected
                    ? "text-accent-primary after:absolute after:inset-x-0 after:bottom-0 after:border-b after:border-accent-strong"
                    : "text-primary hover:text-secondary"
                )
              }
            >
              {label}
              {shown !== undefined && <CountChip count={shown} />}
            </Tab>
          );
        })}
      </TabList>
      <TabPanels className="flex min-h-0 flex-1 flex-col">
        {NOTIFICATION_TABS.map((tab) => (
          <TabPanel key={tab.value} className="flex min-h-0 flex-1 flex-col overflow-y-auto">
            {children}
          </TabPanel>
        ))}
      </TabPanels>
    </TabGroup>
  );
});
