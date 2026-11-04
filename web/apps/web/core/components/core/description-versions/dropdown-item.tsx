/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { ProfileAvatar } from "@/components/assets/profile-image";
import { useTranslation } from "@nerve/i18n";
import type { DescriptionVersionOption } from "./root";
import { CustomMenu } from "@nerve/ui";
import { calculateTimeAgo } from "@nerve/utils";
// hooks
import { useMember } from "@/hooks/store/use-member";

type Props = {
  workspaceID: string | undefined;
  onClick: (versionId: string) => void;
  version: DescriptionVersionOption;
};

export const DescriptionVersionsDropdownItem = observer(function DescriptionVersionsDropdownItem(props: Props) {
  const { onClick, version, workspaceID } = props;
  // store hooks
  const { getUserDetails } = useMember();
  // derived values
  const versionCreator = version.ownerId ? getUserDetails(version.ownerId) : null;
  // translation
  const { t } = useTranslation();

  return (
    <CustomMenu.MenuItem key={version.id} className="flex items-center gap-1" onClick={() => onClick(version.id)}>
      <span className="flex-shrink-0">
        <ProfileAvatar
          alt={versionCreator?.display_name ?? t("common.deactivated_user")}
          fallback={(versionCreator?.display_name ?? t("common.deactivated_user"))?.[0]?.toUpperCase()}
          size="2xs"
          source={{ kind: "member", workspaceID: workspaceID ?? "", memberID: version.ownerId }}
        />
      </span>
      <p className="flex items-center gap-1.5 text-11 text-secondary">
        <span className="font-medium">{versionCreator?.display_name ?? t("common.deactivated_user")}</span>
        <span>{calculateTimeAgo(version.lastSavedAt)}</span>
      </p>
    </CustomMenu.MenuItem>
  );
});
