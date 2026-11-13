/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { CloseOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import { ProfileAvatar } from "@/components/assets/profile-image";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";

type Props = {
  handleRemove: (memberId: string) => void;
  values: readonly string[];
};

/**
 * The members a modules list is filtered by, as its lead or its members, each by his name and removable; one whose
 * account the page no longer knows, by the former member's name.
 */
export const AppliedMembersFilters = observer(function AppliedMembersFilters(props: Props) {
  const { handleRemove, values } = props;
  // hooks
  const { t } = useTranslation();
  const imageWorkspace = useRouteWorkspace();
  const { getUserDetails } = useMember();

  return (
    <>
      {values.map((memberId) => {
        const name = getUserDetails(memberId)?.display_name ?? t("distribution.former_member");
        return (
          <div key={memberId} className="flex items-center gap-1 rounded-sm bg-layer-3 px-1.5 py-1 text-11">
            <ProfileAvatar
              alt={name}
              fallback={name[0]?.toUpperCase()}
              source={{ kind: "member", workspaceID: imageWorkspace?.id ?? "", memberID: memberId }}
              size="2xs"
            />
            <span className="normal-case">{name}</span>
            <button
              type="button"
              className="grid place-items-center text-tertiary hover:text-secondary"
              aria-label={t("common.remove_item", { name })}
              onClick={() => handleRemove(memberId)}
            >
              <CloseOutline height={10} width={10} aria-hidden="true" />
            </button>
          </div>
        );
      })}
    </>
  );
});
