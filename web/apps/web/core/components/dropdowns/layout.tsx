/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { ISSUE_LAYOUT_MAP } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { EIssueLayoutTypes } from "@nerve/types";
import { getButtonStyling } from "@nerve/propel/button";
import { CustomSelect } from "@nerve/ui";
import { IssueLayoutIcon } from "@/components/issues/issue-layouts/layout-icon";

type TLayoutDropDown = {
  onChange: (value: EIssueLayoutTypes) => void;
  value: EIssueLayoutTypes;
  disabledLayouts?: EIssueLayoutTypes[];
};

export const LayoutDropDown = observer(function LayoutDropDown(props: TLayoutDropDown) {
  const { onChange, value = EIssueLayoutTypes.LIST, disabledLayouts = [] } = props;
  const { t } = useTranslation();
  const selected = ISSUE_LAYOUT_MAP[value];
  return (
    <CustomSelect
      value={value}
      onChange={onChange}
      customButtonClassName={getButtonStyling("secondary", "lg")}
      customButton={
        <span className="flex items-center gap-2 text-secondary">
          <IssueLayoutIcon layout={selected.key} strokeWidth={2} className="size-3.5 text-secondary" />
          <span className="text-11 font-medium">{t(selected.i18n_label)}</span>
        </span>
      }
    >
      {Object.values(ISSUE_LAYOUT_MAP)
        .filter((layout) => !disabledLayouts.includes(layout.key))
        .map((layout) => (
          <CustomSelect.Option key={layout.key} value={layout.key}>
            <span className="flex items-center gap-2">
              <IssueLayoutIcon layout={layout.key} strokeWidth={2} className="size-3 text-secondary" />
              <span className="text-11 font-medium">{t(layout.i18n_label)}</span>
            </span>
          </CustomSelect.Option>
        ))}
    </CustomSelect>
  );
});
