/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { ModuleLink } from "@nerve/api-client";
// hooks
import { useModules } from "@/hooks/store/use-modules";
// local imports
import { ModuleLinkItem } from "./list-item";
import { useModuleLinkMetadataRefresh } from "./use-metadata-refresh";

type Props = {
  moduleId: string;
  /** Whether the caller changes and removes the links (the sidebar's root decides it). */
  editable: boolean;
  /** The links whose removal is out. */
  removing: ReadonlySet<string>;
  onEdit: (link: ModuleLink) => void;
  onRemove: (link: ModuleLink) => void;
};

/**
 * A module's links as nerve lists them, the latest added first (P6a spec 2.8): to every caller who reads the module.
 * While they show, the links whose pages nerve has yet to read are read again, within bounds (R10).
 */
export const ModuleLinksList = observer(function ModuleLinksList(props: Props) {
  const { moduleId, editable, removing, onEdit, onRemove } = props;
  // store hooks
  const { getModuleLinks } = useModules();
  useModuleLinkMetadataRefresh(moduleId);
  const links = getModuleLinks(moduleId) ?? [];

  return (
    <ul className="flex flex-col gap-3">
      {links.map((link) => (
        <ModuleLinkItem
          key={link.id}
          link={link}
          editable={editable}
          removing={removing.has(link.id)}
          onEdit={() => onEdit(link)}
          onRemove={() => onRemove(link)}
        />
      ))}
    </ul>
  );
});
