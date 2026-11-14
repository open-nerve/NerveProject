/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";
import type { IssueChange } from "@nerve/types";

import React from "react";
import { observer } from "mobx-react";
// types

// hooks
// components
import { IssuePropertyLabels } from "../../properties";

type Props = {
  issue: Issue;
  onClose: () => void;
  onChange: (issue: Issue, data: IssueChange) => void;
  disabled: boolean;
};

export const SpreadsheetLabelColumn = observer(function SpreadsheetLabelColumn(props: Props) {
  const { issue, onChange, disabled, onClose } = props;
  // hooks

  return (
    <div className="h-11 w-full border-b-[0.5px] border-subtle">
      <IssuePropertyLabels
        issueName={issue.name}
        projectId={issue.project_id ?? null}
        value={issue.label_ids || []}
        onChange={(data) => onChange(issue, { label_ids: data })}
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
