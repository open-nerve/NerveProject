/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ChevronDownOutline, SortAscendingOutline, SortDescendingOutline, TickOutline } from "@makeplane/propel/icons";
// nerve imports
import { MODULE_ORDER_BY_OPTIONS } from "@nerve/constants";
import type { ModuleOrderBy } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getButtonStyling } from "@nerve/propel/button";
import { CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";

/** An option of the list's order, without its direction. */
type OrderKey = (typeof MODULE_ORDER_BY_OPTIONS)[number]["key"];

/** The order by key, the highest first when descending; the manual order has no direction. */
const orderOf = (key: OrderKey, descending: boolean): ModuleOrderBy =>
  descending && key !== "sort_order" ? `-${key}` : key;

type Props = {
  value: ModuleOrderBy;
  onChange: (value: ModuleOrderBy) => void;
};

/**
 * The order of a project's modules list (P6a spec 2.4, R14): a menu of its options, by the pointer or the keyboard
 * (CustomMenu), its button named by the order chosen. An option keeps the direction chosen, the lowest first or the
 * highest first, and the direction is chosen below them; the manual order has none, and an option chosen after it is
 * the lowest first.
 */
export function ModuleOrderByDropdown({ value, onChange }: Props) {
  // hooks
  const { t } = useTranslation();

  const descending = value.startsWith("-");
  const key = (descending ? value.slice(1) : value) as OrderKey;
  const chosen = MODULE_ORDER_BY_OPTIONS.find((option) => option.key === key);
  const label = chosen ? t(chosen.i18n_label) : "";
  const choose = (next: ModuleOrderBy) => {
    if (next !== value) onChange(next);
  };
  const tick = <TickOutline className="h-3 w-3" aria-hidden="true" />;

  return (
    <CustomMenu
      ariaLabel={`${t("common.order_by.label")}: ${label}`}
      customButton={
        <span className={cn(getButtonStyling("secondary", "lg"), "px-2 text-tertiary")}>
          {descending ? (
            <SortDescendingOutline className="size-3" aria-hidden="true" />
          ) : (
            <SortAscendingOutline className="size-3" aria-hidden="true" />
          )}
          {label}
          <ChevronDownOutline className="size-3" aria-hidden="true" />
        </span>
      }
      placement="bottom-end"
      maxHeight="lg"
      closeOnSelect
    >
      {MODULE_ORDER_BY_OPTIONS.map((option) => (
        <CustomMenu.MenuItem
          key={option.key}
          className="flex items-center justify-between gap-2"
          onClick={() => choose(orderOf(option.key, descending))}
        >
          {t(option.i18n_label)}
          {option.key === key && tick}
        </CustomMenu.MenuItem>
      ))}
      {key !== "sort_order" && (
        <>
          <hr className="my-2 border-subtle" />
          <CustomMenu.MenuItem
            className="flex items-center justify-between gap-2"
            onClick={() => choose(orderOf(key, false))}
          >
            {t("common.order_by.asc")}
            {!descending && tick}
          </CustomMenu.MenuItem>
          <CustomMenu.MenuItem
            className="flex items-center justify-between gap-2"
            onClick={() => choose(orderOf(key, true))}
          >
            {t("common.order_by.desc")}
            {descending && tick}
          </CustomMenu.MenuItem>
        </>
      )}
    </CustomMenu>
  );
}
