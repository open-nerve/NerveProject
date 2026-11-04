/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { ProfileAvatar } from "@/components/assets/profile-image";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
import { CloseOutline } from "@makeplane/propel/icons";
// helpers
// types
import { useMember } from "@/hooks/store/use-member";

type Props = {
  handleRemove: (val: string) => void;
  values: string[];
  editable: boolean | undefined;
};

export const AppliedMembersFilters = observer(function AppliedMembersFilters(props: Props) {
  const { handleRemove, values, editable } = props;
  const imageWorkspace = useRouteWorkspace();
  // store hooks
  const {
    workspace: { getWorkspaceMemberDetails },
  } = useMember();
  const { t } = useTranslation();

  return (
    <>
      {values.map((memberId) => {
        const memberDetails = getWorkspaceMemberDetails(memberId)?.member;

        if (!memberDetails) return null;

        return (
          <div key={memberId} className="flex items-center gap-1 rounded-sm bg-layer-1 px-1.5 py-1 text-11">
            <ProfileAvatar
              alt={memberDetails.display_name}
              fallback={memberDetails.display_name?.[0]?.toUpperCase()}
              source={{ kind: "member", workspaceID: imageWorkspace?.id ?? "", memberID: memberId }}
              size="2xs"
            />
            <span className="normal-case">{memberDetails.display_name}</span>
            {editable && (
              <button
                type="button"
                className="grid place-items-center text-tertiary hover:text-secondary"
                onClick={() => handleRemove(memberId)}
                aria-label={t("common.remove_item", { name: memberDetails.display_name })}
              >
                <CloseOutline height={10} width={10} />
              </button>
            )}
          </div>
        );
      })}
    </>
  );
});
