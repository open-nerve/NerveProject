/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useParams } from "react-router";
// components
import type { TPowerKPageType } from "@/components/power-k/core/types";
import { PowerKMembersMenu } from "@/components/power-k/menus/members";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useModules } from "@/hooks/store/use-modules";
// local imports
import { PowerKModuleStatusMenu } from "./status-menu";

type Props = {
  activePage: TPowerKPageType | null;
  handleSelection: (data: unknown) => void;
};

/**
 * The pages of the module's page's commands (commands.tsx): its members, of its project's who are not its guests (as
 * nerve takes a member), each shown chosen while he is one; and its status, the module's shown chosen.
 */
export const PowerKModuleContextBasedPages = observer(function PowerKModuleContextBasedPages(props: Props) {
  const { activePage, handleSelection } = props;
  // navigation
  const { moduleId } = useParams();
  // store hooks
  const { getModuleById } = useModules();
  const {
    project: { getProjectMemberIds },
  } = useMember();
  // derived values
  const module = getModuleById(moduleId);

  if (!module) return null;

  return (
    <>
      {/* members menu */}
      {activePage === "update-module-member" && (
        <PowerKMembersMenu
          handleSelect={handleSelection}
          userIds={getProjectMemberIds(module.project_id, false) ?? undefined}
          value={module.member_ids}
        />
      )}
      {/* status menu */}
      {activePage === "update-module-status" && (
        <PowerKModuleStatusMenu handleSelect={handleSelection} value={module.status} />
      )}
    </>
  );
});
