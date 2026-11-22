/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import type { NotificationTab } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { EmptyStateCompact } from "@nerve/propel/empty-state";

type Props = {
  /** The tab whose list has no notification. */
  tab: NotificationTab;
};

/** That a tab of the notifications page lists no notification (P4 spec 2.6), said as that tab says it. */
export function NotificationEmptyState({ tab }: Props) {
  const { t } = useTranslation();
  return (
    <EmptyStateCompact
      assetKey="inbox"
      assetClassName="size-24"
      title={
        tab === "all"
          ? t("workspace_empty_state.inbox_sidebar_all.title")
          : t("workspace_empty_state.inbox_sidebar_mentions.title")
      }
      className="max-w-56"
    />
  );
}
