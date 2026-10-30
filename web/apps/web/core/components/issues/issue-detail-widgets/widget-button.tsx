/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
// helpers
import { getButtonStyling } from "@nerve/propel/button";

type Props = {
  icon: React.ReactNode;
  title: string;
};

export function IssueDetailWidgetButton(props: Props) {
  const { icon, title } = props;
  return (
    <span className={getButtonStyling("secondary", "lg")}>
      {icon && icon}
      <span className="text-body-xs-medium">{title}</span>
    </span>
  );
}
