/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import React from "react";
import { observer } from "mobx-react";
// types

// components
import { PriorityDropdown } from "@/components/dropdowns/priority";

type Props = {
  issue: Issue;
  onClose: () => void;
  onChange: (issue: Issue, data: Partial<IssueDetail>, updates: any) => void;
  disabled: boolean;
};

export const SpreadsheetPriorityColumn = observer(function SpreadsheetPriorityColumn(props: Props) {
  const { issue, onChange, disabled, onClose } = props;

  return (
    <div className="h-11 border-b-[0.5px] border-subtle">
      <PriorityDropdown
        value={issue.priority}
        onChange={(data) => onChange(issue, { priority: data }, { changed_property: "priority", change_details: data })}
        disabled={disabled}
        buttonVariant="transparent-with-text"
        buttonClassName="text-left rounded-none px-page-x"
        buttonContainerClassName="w-full"
        onClose={onClose}
      />
    </div>
  );
});
