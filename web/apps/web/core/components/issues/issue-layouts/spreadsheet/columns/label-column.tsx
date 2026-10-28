/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import React from "react";
import { observer } from "mobx-react";
// types

// hooks
import { useLabel } from "@/hooks/store/use-label";
// components
import { IssuePropertyLabels } from "../../properties";

type Props = {
  issue: Issue;
  onClose: () => void;
  onChange: (issue: Issue, data: Partial<IssueDetail>, updates: any) => void;
  disabled: boolean;
};

export const SpreadsheetLabelColumn = observer(function SpreadsheetLabelColumn(props: Props) {
  const { issue, onChange, disabled, onClose } = props;
  // hooks
  const { labelMap } = useLabel();

  const defaultLabelOptions =
    issue?.label_ids?.flatMap((id) => {
      const label = labelMap[id];
      return label ? [label] : [];
    }) || [];

  return (
    <div className="h-11 w-full border-b-[0.5px] border-subtle">
      <IssuePropertyLabels
        projectId={issue.project_id ?? null}
        value={issue.label_ids || []}
        defaultOptions={defaultLabelOptions}
        onChange={(data) => onChange(issue, { label_ids: data }, { changed_property: "labels", change_details: data })}
        className="h-full w-full"
        buttonClassName="px-page-x w-full h-full rounded-none"
        hideDropdownArrow
        maxRender={1}
        disabled={disabled}
        placeholderText="Select labels"
        onClose={onClose}
        noLabelBorder
        fullWidth
        fullHeight
      />
    </div>
  );
});
