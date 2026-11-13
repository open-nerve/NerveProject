/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { DATE_AFTER_FILTER_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { CloseOutline } from "@makeplane/propel/icons";
import { renderFormattedDate, capitalizeFirstLetter } from "@nerve/utils";

type Props = {
  handleRemove: (option: string) => void;
  values: readonly string[];
};

/** An option of a day's filter as the menu names it: some time from now, or "After"/"Before" a day of the calendar. */
const dateLabel = (value: string): string => {
  const option = DATE_AFTER_FILTER_OPTIONS.find((d) => d.value === value);
  if (option) return option.name;
  const [day, operator] = value.split(";");
  return day && operator ? `${capitalizeFirstLetter(operator)} ${renderFormattedDate(day)}` : "";
};

/** The options a day of a cycles or a modules list is filtered by, each removable. */
export const AppliedDateFilters = observer(function AppliedDateFilters(props: Props) {
  const { handleRemove, values } = props;
  const { t } = useTranslation();

  return (
    <>
      {values.map((date) => (
        <div key={date} className="flex items-center gap-1 rounded-sm bg-layer-3 px-1.5 py-1 text-11">
          <span className="normal-case">{dateLabel(date)}</span>
          <button
            type="button"
            className="grid place-items-center text-tertiary hover:text-secondary"
            aria-label={t("common.remove_item", { name: dateLabel(date) })}
            onClick={() => handleRemove(date)}
          >
            <CloseOutline height={10} width={10} aria-hidden="true" />
          </button>
        </div>
      ))}
    </>
  );
});
