/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { LucideIcon } from "lucide-react";
// plane imports
import type { TFavoriteEntityType, TLogoProps } from "@nerve/types";
import { FavoriteFolderIcon } from "@nerve/propel/icons";
import { CyclesOutline, ModuleOutline, ProjectsOutline, ViewsOutline } from "@makeplane/propel/icons";
import type { ISvgIcons } from "@nerve/propel/icons";
import { Logo } from "@nerve/propel/emoji-icon-picker";

const ICON_MAP: Record<TFavoriteEntityType, React.FC<ISvgIcons> | LucideIcon> = {
  project: ProjectsOutline,
  view: ViewsOutline,
  module: ModuleOutline,
  cycle: CyclesOutline,
  folder: FavoriteFolderIcon,
};

type Props = {
  type: TFavoriteEntityType;
  logo?: TLogoProps;
};

export const FavoriteItemIcon = ({ type, logo }: Props) => {
  const Icon = ICON_MAP[type];

  return (
    <>
      <div className="hidden size-5 items-center justify-center group-hover:flex">
        <Icon className="m-auto size-4 flex-shrink-0 stroke-[1.5]" />
      </div>
      <div className="flex size-5 items-center justify-center group-hover:hidden">
        {logo?.in_use ? (
          <Logo logo={logo} size={16} type={type === "project" ? "material" : "lucide"} />
        ) : (
          <Icon className="m-auto size-4 flex-shrink-0 stroke-[1.5]" />
        )}
      </div>
    </>
  );
};
