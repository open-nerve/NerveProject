/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { CustomSearchSelect } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import useTimezone from "@/hooks/use-timezone";

type TTimezoneSelect = {
  value: string | undefined;
  onChange: (value: string) => void;
  error?: boolean;
  buttonClassName?: string;
  disabled?: boolean;
};

export const TimezoneSelect = observer(function TimezoneSelect(props: TTimezoneSelect) {
  // props
  const { value, onChange, error = false, buttonClassName = "", disabled = false } = props;
  // hooks
  const { disabled: isDisabled, timezones, selectedValue } = useTimezone();
  const { t } = useTranslation();

  return (
    <div>
      <CustomSearchSelect
        value={value}
        // a zone the list lacks, or any zone while the list loads, shows by its name
        label={value ? (selectedValue(value) ?? value) : t("select_a_timezone")}
        options={isDisabled || disabled ? [] : timezones}
        onChange={onChange}
        buttonClassName={cn(buttonClassName, "border border-subtle-1", {
          "border-danger-strong": error,
        })}
        className="rounded-md"
        optionsClassName="w-72"
        input
        disabled={isDisabled || disabled}
        placement="bottom-end"
        searchPlaceholder={t("common.search.label")}
        noResultsMessage={t("common.search.no_matches_found")}
      />
    </div>
  );
});
