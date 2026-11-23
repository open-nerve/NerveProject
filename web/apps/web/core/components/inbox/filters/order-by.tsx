/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { ChevronDownOutline, SortAscendingOutline, SortDescendingOutline, TickOutline } from "@makeplane/propel/icons";
// nerve imports
import { INTAKE_ORDER_FIELDS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getButtonStyling } from "@nerve/propel/button";
import { CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// store
import type { IntakeFilters } from "@/store/issue/intake/filters";
// local imports
import { useIntakePageState } from "../page-state";

/** A field the list is ordered by (INTAKE_ORDER_FIELDS), without its direction. */
type OrderField = (typeof INTAKE_ORDER_FIELDS)[number]["value"];

/** The order of nerve's six that orders by the field, the highest first when descending (C4). */
const orderOf = (field: OrderField, descending: boolean): IntakeFilters["order"] => (descending ? `-${field}` : field);

/** The field and the direction of an order of nerve's. */
const partsOf = (order: IntakeFilters["order"]) => {
  const descending = order.startsWith("-");
  return { field: (descending ? order.slice(1) : order) as OrderField, descending };
};

/**
 * The order of the list of the intake page's tab (P5a spec 2.2 C4, 2.7): a menu (CustomMenu) of the fields, each
 * keeping the direction chosen, then the two directions, each keeping the field; the two make one of nerve's six
 * orders, the page's (setFilters), the newest first until the user chooses. Its button is named by both.
 */
export const IntakeOrderBy = observer(function IntakeOrderBy() {
  const { t } = useTranslation();
  const { filters, setFilters } = useIntakePageState();
  const { field, descending } = partsOf(filters.order);
  const fieldLabel = t(INTAKE_ORDER_FIELDS.find((option) => option.value === field)?.i18n_label ?? "");
  const choose = (order: IntakeFilters["order"]) => {
    if (order !== filters.order) setFilters((current) => ({ ...current, order }));
  };
  const tick = <TickOutline className="h-3 w-3" aria-hidden="true" />;

  return (
    <CustomMenu
      ariaLabel={t("inbox_issue.a11y.order_by", {
        field: fieldLabel,
        direction: t(descending ? "common.order_by.desc" : "common.order_by.asc"),
      })}
      customButton={
        <span className={cn(getButtonStyling("secondary", "lg"), "px-2 text-tertiary")}>
          {descending ? (
            <SortDescendingOutline className="size-3" aria-hidden="true" />
          ) : (
            <SortAscendingOutline className="size-3" aria-hidden="true" />
          )}
          <span className="hidden xl:inline">{fieldLabel}</span>
          <ChevronDownOutline className="size-3" aria-hidden="true" />
        </span>
      }
      placement="bottom-end"
      maxHeight="lg"
      closeOnSelect
    >
      {INTAKE_ORDER_FIELDS.map((option) => (
        <CustomMenu.MenuItem
          key={option.value}
          className="flex items-center justify-between gap-2"
          onClick={() => choose(orderOf(option.value, descending))}
        >
          {t(option.i18n_label)}
          {option.value === field && tick}
        </CustomMenu.MenuItem>
      ))}
      <hr className="my-2 border-subtle" />
      <CustomMenu.MenuItem
        className="flex items-center justify-between gap-2"
        onClick={() => choose(orderOf(field, false))}
      >
        {t("common.order_by.asc")}
        {!descending && tick}
      </CustomMenu.MenuItem>
      <CustomMenu.MenuItem
        className="flex items-center justify-between gap-2"
        onClick={() => choose(orderOf(field, true))}
      >
        {t("common.order_by.desc")}
        {descending && tick}
      </CustomMenu.MenuItem>
    </CustomMenu>
  );
});
