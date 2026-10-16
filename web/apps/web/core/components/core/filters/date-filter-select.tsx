/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { CalendarOutline } from "@makeplane/propel/icons";
// ui
import { useTranslation } from "@nerve/i18n";
import { CalendarAfterIcon, CalendarBeforeIcon } from "@nerve/propel/icons";
import { CustomSelect } from "@nerve/ui";

type Props = {
  title: string;
  value: string;
  onChange: (value: string) => void;
};

type DueDate = {
  // the filter's sentence, the title of the date filtered in it ({title})
  i18n_name: string;
  value: string;
  icon: React.ReactNode;
};

const dueDateRange: DueDate[] = [
  {
    i18n_name: "date_filter.before",
    value: "before",
    icon: <CalendarBeforeIcon className="h-4 w-4" />,
  },
  {
    i18n_name: "date_filter.after",
    value: "after",
    icon: <CalendarAfterIcon className="h-4 w-4" />,
  },
  {
    i18n_name: "date_filter.range",
    value: "range",
    icon: <CalendarOutline className="h-4 w-4" />,
  },
];

export function DateFilterSelect({ title, value, onChange }: Props) {
  const { t } = useTranslation();
  const selected = dueDateRange.find((item) => item.value === value);
  return (
    <CustomSelect
      value={value}
      label={
        <div className="flex items-center gap-2 text-11">
          {selected?.icon}
          <span>{selected && t(selected.i18n_name, { title })}</span>
        </div>
      }
      onChange={onChange}
    >
      {dueDateRange.map((option) => (
        <CustomSelect.Option key={option.value} value={option.value}>
          <div className="flex items-center gap-2">
            <span>{option.icon}</span>
            {t(option.i18n_name, { title })}
          </div>
        </CustomSelect.Option>
      ))}
    </CustomSelect>
  );
}
