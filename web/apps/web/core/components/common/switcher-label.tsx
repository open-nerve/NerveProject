/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FC } from "react";
import type { LogoProps } from "@nerve/api-client";
import { Logo } from "@nerve/propel/emoji-icon-picker";
import type { ISvgIcons } from "@nerve/propel/icons";
import { truncateText } from "@nerve/utils";

type TSwitcherIconProps = {
  logo_props?: LogoProps;
  LabelIcon: FC<ISvgIcons>;
  size?: number;
  type?: "lucide" | "material";
};

export function SwitcherIcon({ logo_props, LabelIcon, size = 12, type = "lucide" }: TSwitcherIconProps) {
  if (logo_props?.in_use) {
    return <Logo logo={logo_props} size={size} type={type} />;
  }

  return <LabelIcon height={size} width={size} />;
}

type TSwitcherLabelProps = {
  logo_props?: LogoProps;
  name?: string;
  LabelIcon: FC<ISvgIcons>;
  type?: "lucide" | "material";
};

export function SwitcherLabel(props: TSwitcherLabelProps) {
  const { logo_props, name, LabelIcon, type = "lucide" } = props;
  return (
    <div className="flex items-center gap-1 text-secondary">
      <SwitcherIcon logo_props={logo_props} LabelIcon={LabelIcon} type={type} />
      {truncateText(name ?? "", 40)}
    </div>
  );
}
