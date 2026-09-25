/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { CSSProperties, ComponentType, ReactNode } from "react";

// ============================================================
// Chart Base
// ============================================================
export type TChartLegend = {
  align: "left" | "center" | "right";
  verticalAlign: "top" | "middle" | "bottom";
  layout: "horizontal" | "vertical";
  wrapperStyles?: CSSProperties;
};

type TChartMargin = {
  top?: number;
  right?: number;
  bottom?: number;
  left?: number;
};

export type TChartData<K extends string, T extends string> = {
  // required key
  [key in K]: string | number;
} & Record<T, any>;

type TBaseChartProps<K extends string, T extends string> = {
  data: TChartData<K, T>[];
  className?: string;
  legend?: TChartLegend;
  margin?: TChartMargin;
  showTooltip?: boolean;
  customTooltipContent?: (props: { active?: boolean; label: string; payload: any }) => ReactNode;
};

// Props specific to charts with X and Y axes
type TAxisChartProps<K extends string, T extends string> = TBaseChartProps<K, T> & {
  xAxis: {
    key: keyof TChartData<K, T>;
    label?: string;
    strokeColor?: string;
    dy?: number;
  };
  yAxis: {
    allowDecimals?: boolean;
    domain?: [number, number];
    key: keyof TChartData<K, T>;
    label?: string;
    strokeColor?: string;
    offset?: number;
    dx?: number;
  };
  tickCount?: {
    x?: number;
    y?: number;
  };
  customTicks?: {
    x?: ComponentType<unknown>;
    y?: ComponentType<unknown>;
  };
};

// ============================================================
// Area Chart
// ============================================================

type TAreaItem<T extends string> = {
  key: T;
  label: string;
  stackId: string;
  fill: string;
  fillOpacity: number;
  showDot: boolean;
  smoothCurves: boolean;
  strokeColor: string;
  strokeOpacity: number;
  style?: Record<string, string | number>;
};

export type TAreaChartProps<K extends string, T extends string> = TAxisChartProps<K, T> & {
  areas: TAreaItem<T>[];
  comparisonLine?: {
    dashedLine: boolean;
    strokeColor: string;
  };
};
