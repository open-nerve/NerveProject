/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Dispatch, SetStateAction } from "react";
import { useState } from "react";
import { CloseOutline, EditOutline } from "@makeplane/propel/icons";
// types
import type { Label } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
// components
import { CreateUpdateLabelInline } from "./create-update-label-inline";
import type { ICustomMenuItem } from "./label-block/label-item-block";
import { LabelItemBlock } from "./label-block/label-item-block";
import { LabelDndHOC } from "./label-drag-n-drop-HOC";
import { useLabelChanges } from "./use-label-changes";

type Props = {
  label: Label;
  handleLabelDelete: (label: Label) => void;
  setIsUpdating: Dispatch<SetStateAction<boolean>>;
  isParentDragging?: boolean;
  isChild: boolean;
  isLastChild: boolean;
  onDrop: (
    draggingLabelId: string,
    droppedParentId: string | null,
    droppedLabelId: string | undefined,
    dropAtEndOfList: boolean
  ) => void;
  isEditable?: boolean;
};

export function ProjectSettingLabelItem(props: Props) {
  const {
    label,
    setIsUpdating,
    handleLabelDelete,
    isChild,
    isLastChild,
    isParentDragging = false,
    onDrop,
    isEditable = false,
  } = props;
  // states
  const [isEditLabelForm, setEditLabelForm] = useState(false);
  const { removeFromGroup } = useLabelChanges();
  const { t } = useTranslation();

  const customMenuItems: ICustomMenuItem[] = [
    {
      CustomIcon: CloseOutline,
      onClick: () => void removeFromGroup(label.id),
      isVisible: label.parent_id !== null,
      text: t("project_settings.labels.remove_from_group"),
      key: "remove_from_group",
    },
    {
      CustomIcon: EditOutline,
      onClick: () => {
        setEditLabelForm(true);
        setIsUpdating(true);
      },
      isVisible: true,
      text: t("project_settings.labels.edit"),
      key: "edit_label",
    },
  ];

  return (
    <LabelDndHOC label={label} isGroup={false} isChild={isChild} isLastChild={isLastChild} onDrop={onDrop}>
      {(isDragging, isDroppingInLabel, dragHandleRef) => (
        <div
          className={`rounded-sm ${isDroppingInLabel ? "border-[2px] border-accent-strong" : "border-[1.5px] border-transparent"}`}
        >
          <div
            className={`group relative flex items-center justify-between gap-2 space-y-3 rounded-sm px-1 py-3 ${
              isDroppingInLabel ? "" : "border-[0.5px] border-subtle"
            } ${isDragging || isParentDragging ? "bg-layer-1" : "bg-surface-1"}`}
          >
            {isEditLabelForm ? (
              <CreateUpdateLabelInline
                projectId={label.project_id}
                labelToUpdate={label}
                onClose={() => {
                  setEditLabelForm(false);
                  setIsUpdating(false);
                }}
              />
            ) : (
              <LabelItemBlock
                label={label}
                isDragging={isDragging}
                customMenuItems={customMenuItems}
                handleLabelDelete={handleLabelDelete}
                dragHandleRef={dragHandleRef}
                disabled={!isEditable}
              />
            )}
          </div>
        </div>
      )}
    </LabelDndHOC>
  );
}
