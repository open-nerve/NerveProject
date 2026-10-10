/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Controller, useFormContext } from "react-hook-form";
// nerve imports
import { EUserPermissions, NETWORK_CHOICES, ETabIndices } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { CustomSelect } from "@nerve/ui";
import { getTabIndex } from "@nerve/utils";
// components
import { MemberDropdownBase } from "@/components/dropdowns/member/base";
import { ProjectNetworkIcon } from "@/components/project/project-network-icon";
// hooks
import { useMember } from "@/hooks/store/use-member";
// local imports
import type { ProjectCreationForm } from "./use-create-project";

type Props = {
  isMobile?: boolean;
};

const ProjectAttributes = observer(function ProjectAttributes(props: Props) {
  const { isMobile = false } = props;
  const { t } = useTranslation();
  const { control } = useFormContext<ProjectCreationForm>();
  const { getIndex } = getTabIndex(ETabIndices.PROJECT_CREATE, isMobile);
  const {
    getUserDetails,
    workspace: { workspaceMemberIds, getWorkspaceMemberDetails },
  } = useMember();
  // the lead is one of the workspace's admins and members (M3 design 3.19): no guest, no one whose membership ended
  const leadCandidates = (workspaceMemberIds ?? []).filter((id) => {
    const membership = getWorkspaceMemberDetails(id);
    return membership !== null && membership.is_active && membership.role >= EUserPermissions.MEMBER;
  });
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Controller
        name="network"
        control={control}
        render={({ field: { onChange, value } }) => {
          const currentNetwork = NETWORK_CHOICES.find((n) => n.key === value);

          return (
            <div className="h-7 flex-shrink-0" tabIndex={getIndex("network")}>
              <CustomSelect
                value={value}
                onChange={onChange}
                label={
                  <div className="flex h-full items-center gap-1">
                    {currentNetwork ? (
                      <>
                        <ProjectNetworkIcon iconKey={currentNetwork.iconKey} />
                        {t(currentNetwork.i18n_label)}
                      </>
                    ) : (
                      <span className="text-placeholder">{t("select_network")}</span>
                    )}
                  </div>
                }
                placement="bottom-start"
                className="h-full"
                buttonClassName="h-full"
                noChevron
                tabIndex={getIndex("network")}
              >
                {NETWORK_CHOICES.map((network) => (
                  <CustomSelect.Option key={network.key} value={network.key}>
                    <div className="flex items-start gap-2">
                      <ProjectNetworkIcon iconKey={network.iconKey} className="h-3.5 w-3.5" />
                      <div className="-mt-1">
                        <p>{t(network.i18n_label)}</p>
                        <p className="text-11 text-placeholder">{t(network.description)}</p>
                      </div>
                    </div>
                  </CustomSelect.Option>
                ))}
              </CustomSelect>
            </div>
          );
        }}
      />
      <Controller
        name="project_lead_id"
        control={control}
        render={({ field: { value, onChange } }) => (
          <div className="h-7 flex-shrink-0">
            <MemberDropdownBase
              getUserDetails={getUserDetails}
              memberIds={leadCandidates}
              value={value}
              // the lead picked again is no lead
              onChange={(lead) => onChange(lead === value ? null : lead)}
              placeholder={t("lead")}
              multiple={false}
              buttonVariant="border-with-text"
              tabIndex={getIndex("lead")}
            />
          </div>
        )}
      />
    </div>
  );
});

export { ProjectAttributes };
