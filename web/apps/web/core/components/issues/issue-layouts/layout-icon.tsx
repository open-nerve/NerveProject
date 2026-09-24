/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { BoardOutline, CalendarOutline, ListOutline, TableOutline } from "@makeplane/propel/icons";
import type { ISvgIcons } from "@nerve/propel/icons";
import { EIssueLayoutTypes } from "@nerve/types";

export function IssueLayoutIcon({
  layout,
  size,
  ...props
}: { layout: EIssueLayoutTypes; size?: number } & Omit<ISvgIcons, "width" | "height">) {
  const iconProps = {
    ...props,
    ...(size && { width: size, height: size }),
  };

  switch (layout) {
    case EIssueLayoutTypes.LIST:
      return <ListOutline {...iconProps} />;
    case EIssueLayoutTypes.KANBAN:
      return <BoardOutline {...iconProps} />;
    case EIssueLayoutTypes.CALENDAR:
      return <CalendarOutline {...iconProps} />;
    case EIssueLayoutTypes.SPREADSHEET:
      return <TableOutline {...iconProps} />;
    default:
      return null;
  }
}
