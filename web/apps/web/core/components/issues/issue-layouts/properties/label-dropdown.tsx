/** Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only. See the LICENSE file for details. */
import type { Placement } from "@popperjs/core";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { ChevronDownOutline } from "@makeplane/propel/icons";
import { EUserPermissionsLevel } from "@nerve/constants";
import { EUserProjectRoles } from "@nerve/types";
import { cn } from "@nerve/utils";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useLabel } from "@/hooks/store/use-label";
import { useUserPermissions } from "@/hooks/store/user";
import { useLabelChanges } from "@/components/labels/use-label-changes";
import { DirectoryReadError, useDirectoryRead } from "../../issue-detail-widgets/directory-read";
import { useDetailScope } from "../../issue-detail-widgets/use-detail-scope";
import { useIssueLabels } from "../../use-issue-labels";
import { WorkItemLabelSelectBase } from "../../select/base";

export interface ILabelDropdownProps {
  projectId: string | null;
  issueName?: string;
  value: string[];
  onChange: (data: string[]) => void;
  onClose?: () => void;
  disabled?: boolean;
  hideDropdownArrow?: boolean;
  className?: string;
  buttonClassName?: string;
  optionsClassName?: string;
  placement?: Placement;
  maxRender?: number;
  renderByDefault?: boolean;
  fullWidth?: boolean;
  fullHeight?: boolean;
  label: React.ReactNode;
}

export const LabelDropdown = observer(function LabelDropdown(props: ILabelDropdownProps) {
  const { workspaceSlug } = useParams();
  return <LabelDropdownContent key={JSON.stringify([workspaceSlug, props.projectId])} {...props} />;
});
const LabelDropdownContent = observer(function LabelDropdownContent(props: ILabelDropdownProps) {
  const {
    projectId,
    issueName,
    value,
    onChange,
    onClose,
    disabled,
    hideDropdownArrow = false,
    className,
    buttonClassName,
    optionsClassName,
    placement,
    maxRender = 2,
    fullWidth = false,
    fullHeight = false,
    label,
  } = props;
  const { workspaceSlug } = useParams();
  const { fetchProjectLabels, getLabelById } = useLabel();
  const labels = useIssueLabels(workspaceSlug, projectId);
  const { allowPermissions } = useUserPermissions();
  const capture = useDetailScope(workspaceSlug ?? "", projectId ?? undefined, "label-create", true);
  const { create } = useLabelChanges(capture);
  const refuse = useRefusalToast();
  const select = async (ids: string[]) => {
    const current = capture();
    try {
      await onChange(ids);
    } catch (error) {
      if (!current()) throw new SessionChangedError();
      throw error;
    }
  };
  const read = useDirectoryRead(
    workspaceSlug,
    projectId ?? undefined,
    projectId ? () => fetchProjectLabels(projectId) : undefined
  );
  const canCreate =
    !!projectId && allowPermissions([EUserProjectRoles.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, projectId);
  return (
    <WorkItemLabelSelectBase
      grouped={false}
      issueName={issueName}
      value={value}
      onChange={select}
      onChangeError={(error) => {
        if (!(error instanceof SessionChangedError)) refuse(error);
      }}
      onClose={onClose}
      disabled={disabled}
      createLabelEnabled={canCreate}
      getLabelById={getLabelById}
      labelIds={labels?.map((item) => item.id) ?? []}
      onDropdownOpen={read.load}
      isInitializing={read.loading}
      readStatus={read.error ? <DirectoryReadError error={read.error} retry={read.load} /> : undefined}
      onCreateLabel={async (data, underField, done) => {
        if (projectId) await create(projectId, data, underField, done);
      }}
      className={cn(fullHeight ? "h-full" : "h-5", className)}
      optionsClassName={optionsClassName}
      placement={placement}
      buttonContainerClassName={cn(
        "clickable w-full justify-center gap-1 text-caption-sm-regular",
        (fullWidth || value.length > maxRender) && "hover:bg-layer-1",
        disabled ? "cursor-not-allowed text-secondary" : "cursor-pointer",
        buttonClassName
      )}
      label={
        <>
          {label}
          {!hideDropdownArrow && !disabled && <ChevronDownOutline className="h-3 w-3" aria-hidden="true" />}
        </>
      }
    />
  );
});
