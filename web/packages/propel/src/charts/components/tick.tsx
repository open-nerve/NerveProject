/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/* eslint-disable @typescript-eslint/no-explicit-any */
import React from "react";

// Common classnames
const AXIS_TICK_CLASSNAME = "fill-tertiary text-13";

export const CustomXAxisTick = React.memo(function CustomXAxisTick({ x, y, payload, getLabel }: any) {
  return (
    <g transform={`translate(${x},${y})`}>
      <text y={0} dy={16} textAnchor="middle" className={AXIS_TICK_CLASSNAME}>
        {getLabel ? getLabel(payload.value) : payload.value}
      </text>
    </g>
  );
});
CustomXAxisTick.displayName = "CustomXAxisTick";

export const CustomYAxisTick = React.memo(function CustomYAxisTick({ x, y, payload }: any) {
  return (
    <g transform={`translate(${x},${y})`}>
      <text dx={-10} textAnchor="middle" className={AXIS_TICK_CLASSNAME}>
        {payload.value}
      </text>
    </g>
  );
});

CustomYAxisTick.displayName = "CustomYAxisTick";
