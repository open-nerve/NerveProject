/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Command } from "cmdk";
import { observer } from "mobx-react";

import { useRouteWorkspace } from "@/hooks/use-route-workspace";
import { ProfileAvatar } from "@/components/assets/profile-image";
// hooks
import { useMember } from "@/hooks/store/use-member";
// local imports
import { PowerKModalCommandItem } from "../ui/modal/command-item";

type Props = {
  handleSelect: (assigneeId: string) => void;
  userIds: string[] | undefined;
  value: string[];
};

export const PowerKMembersMenu = observer(function PowerKMembersMenu(props: Props) {
  const { handleSelect, userIds, value } = props;
  const imageWorkspace = useRouteWorkspace();
  // store hooks
  const { getUserDetails } = useMember();

  return (
    <Command.Group>
      {userIds?.map((memberId) => {
        const memberDetails = getUserDetails(memberId);
        if (!memberDetails) return;

        return (
          <PowerKModalCommandItem
            key={memberId}
            iconNode={
              <ProfileAvatar
                alt={memberDetails?.display_name}
                fallback={memberDetails?.display_name?.[0]?.toUpperCase()}
                source={{ kind: "member", workspaceID: imageWorkspace?.id ?? "", memberID: memberId }}
                size="xs"
              />
            }
            isSelected={value.includes(memberId)}
            label={memberDetails?.display_name}
            onSelect={() => handleSelect(memberId)}
          />
        );
      })}
    </Command.Group>
  );
});
