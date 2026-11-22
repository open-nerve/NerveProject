/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { PillButton } from "@makeplane/propel/components/pill";
import { CloseOutline } from "@makeplane/propel/icons";
// nerve imports
import { NOTIFICATION_TYPE_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// local imports
import { useNotificationPageState } from "../page-state";

/**
 * The senders the notifications page's all tab is filtered by (P4 spec 2.6, R10): a pill for each, in the type filter's
 * order, a button named as removing it, and clear all. Nothing while none is chosen, nor in the mentions tab, whose
 * list takes no type: the all tab's choice stays for its return.
 */
export const NotificationAppliedFilters = observer(function NotificationAppliedFilters() {
  const { t } = useTranslation();
  const { filters, toggleType, clearTypes } = useNotificationPageState();
  if (filters.tab !== "all" || filters.types.length === 0) return null;
  return (
    <div className="flex flex-shrink-0 flex-wrap items-center gap-2 border-b border-subtle px-3 py-2">
      {NOTIFICATION_TYPE_OPTIONS.filter((option) => filters.types.includes(option.value)).map((option) => {
        const label = t(option.i18n_label);
        return (
          <PillButton
            key={option.value}
            type="button"
            size="md"
            variant="outline"
            label={label}
            aria-label={t("common.remove_item", { name: label })}
            endIcon={<CloseOutline height={12} width={12} aria-hidden="true" />}
            onClick={() => toggleType(option.value)}
          />
        );
      })}
      <PillButton
        type="button"
        size="md"
        variant="outline"
        label={t("common.clear_all")}
        endIcon={<CloseOutline height={12} width={12} aria-hidden="true" />}
        onClick={clearTypes}
      />
    </div>
  );
});
