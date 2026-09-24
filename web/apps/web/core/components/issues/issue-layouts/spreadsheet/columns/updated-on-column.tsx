/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
// types
import type { TIssue } from "@nerve/types";
// helpers
import { Row } from "@nerve/ui";
import { renderFormattedDate } from "@nerve/utils";

type Props = {
  issue: TIssue;
};

export const SpreadsheetUpdatedOnColumn = observer(function SpreadsheetUpdatedOnColumn(props: Props) {
  const { issue } = props;

  return (
    <Row className="flex h-11 w-full items-center border-b-[0.5px] border-subtle-1 text-11 hover:bg-layer-1">
      {renderFormattedDate(issue.updated_at)}
    </Row>
  );
});
