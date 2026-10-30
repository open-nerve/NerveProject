/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
// nerve helpers
import { useOutsideClickDetector } from "@nerve/hooks";
// i18n
import { useTranslation } from "@nerve/i18n";
import { LabelsOutline } from "@makeplane/propel/icons";
// types
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { Label } from "@nerve/api-client";
// hooks
import { cn } from "@nerve/utils";
import { useParams } from "react-router";
import { useIssueLabels } from "../../use-issue-labels";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { LabelDropdown } from "./label-dropdown";

export interface IIssuePropertyLabels {
  projectId: string | null;
  value: string[];
  onChange: (data: string[]) => void;
  disabled?: boolean;
  hideDropdownArrow?: boolean;
  className?: string;
  buttonClassName?: string;
  maxRender?: number;
  noLabelBorder?: boolean;
  placeholderText?: string;
  onClose?: () => void;
  renderByDefault?: boolean;
  fullWidth?: boolean;
  fullHeight?: boolean;
}

type NoLabelProps = {
  isMobile: boolean;
  noLabelBorder: boolean;
  fullWidth: boolean;
  placeholderText?: string;
};

const NoLabel = observer(function NoLabel({ isMobile, noLabelBorder, fullWidth, placeholderText }: NoLabelProps) {
  const { t } = useTranslation();

  return (
    <Tooltip label={`${t("common.labels")}: None`} disabled={isMobile}>
      <div
        className={cn(
          "flex h-full items-center justify-center gap-2 rounded-sm px-2.5 py-1 text-caption-sm-regular hover:bg-layer-1",
          noLabelBorder ? "rounded-none" : "border-[0.5px] border-strong",
          fullWidth && "w-full"
        )}
      >
        <LabelsOutline className="h-3.5 w-3.5" />
        {placeholderText}
      </div>
    </Tooltip>
  );
});

type LabelSummaryProps = {
  isMobile: boolean;
  fullWidth: boolean;
  noLabelBorder: boolean;
  disabled?: boolean;
  projectLabels: Label[];
  value: string[];
};

function LabelSummary({ isMobile, fullWidth, noLabelBorder, disabled, projectLabels, value }: LabelSummaryProps) {
  const { t } = useTranslation();
  // single pass: collect the selected label names instead of filter -> map -> join
  const selectedLabelNames = projectLabels.reduce<string[]>((names, label) => {
    if (value.includes(label?.id)) names.push(label?.name);
    return names;
  }, []);
  return (
    <div
      className={cn(
        "flex h-5 flex-shrink-0 items-center justify-center rounded-sm px-2.5 text-caption-sm-regular",
        fullWidth && "w-full",
        noLabelBorder ? "rounded-none" : "border-[0.5px] border-strong",
        disabled ? "cursor-not-allowed" : "cursor-pointer"
      )}
    >
      <Tooltip label={`${t("common.labels")}: ${selectedLabelNames.join(", ")}`} layout="stacked" disabled={isMobile}>
        <div className="flex h-full items-center gap-1.5 text-secondary">
          <span className="h-2 w-2 flex-shrink-0 rounded-full bg-accent-primary" />
          {`${value.length} Labels`}
        </div>
      </Tooltip>
    </div>
  );
}

type LabelItemProps = {
  label: Label;
  isMobile: boolean;
  renderByDefault: boolean;
  disabled?: boolean;
  fullWidth: boolean;
  noLabelBorder: boolean;
};

const LabelItem = observer(function LabelItem({ label, isMobile, disabled, fullWidth, noLabelBorder }: LabelItemProps) {
  const { t } = useTranslation();

  return (
    <Tooltip label={`${t("common.labels")}: ${label?.name ?? ""}`} layout="stacked" disabled={isMobile}>
      <div
        className={cn(
          "flex h-full max-w-full flex-shrink-0 items-center justify-center overflow-hidden rounded-sm px-2.5 text-caption-sm-regular hover:bg-layer-1",
          !disabled && "cursor-pointer",
          fullWidth && "w-full",
          noLabelBorder ? "rounded-none" : "border-[0.5px] border-strong"
        )}
      >
        <div className="flex max-w-full items-center gap-1.5 overflow-hidden text-secondary">
          <span
            className="h-2 w-2 flex-shrink-0 rounded-full"
            style={{
              backgroundColor: label?.color ?? "#000000",
            }}
          />
          <div className="line-clamp-1 inline-block w-auto max-w-[200px] truncate">{label?.name}</div>
        </div>
      </div>
    </Tooltip>
  );
});

export const IssuePropertyLabels = observer(function IssuePropertyLabels(props: IIssuePropertyLabels) {
  const {
    projectId,
    value,
    onChange,
    onClose,
    disabled,
    hideDropdownArrow = false,
    buttonClassName = "",
    maxRender = 2,
    noLabelBorder = false,
    placeholderText,
    renderByDefault = true,
    fullWidth = false,
    fullHeight = false,
  } = props;
  // states
  const [isOpen, setIsOpen] = useState(false);
  // refs
  const dropdownRef = useRef<HTMLDivElement | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  // store hooks
  const { workspaceSlug } = useParams();
  const { isMobile } = usePlatformOS();
  const storeLabels = useIssueLabels(workspaceSlug, projectId);

  const handleClose = () => {
    if (!isOpen) return;
    setIsOpen(false);
    if (onClose) onClose();
  };

  useOutsideClickDetector(dropdownRef, handleClose);

  useEffect(() => {
    if (isOpen && inputRef.current && !isMobile) {
      inputRef.current.focus();
    }
  }, [isOpen, isMobile]);

  const projectLabels = storeLabels ?? [];

  return (
    <>
      {value.length > 0 ? (
        value.length <= maxRender ? (
          projectLabels
            ?.filter((l) => value.includes(l?.id))
            .map((label) => (
              <LabelDropdown
                disabled={disabled}
                key={label.id}
                projectId={projectId}
                value={value}
                onChange={onChange}
                buttonClassName={buttonClassName}
                hideDropdownArrow={hideDropdownArrow}
                fullWidth={fullWidth}
                fullHeight={fullHeight}
                label={
                  <LabelItem
                    label={label}
                    isMobile={isMobile}
                    renderByDefault={renderByDefault}
                    disabled={disabled}
                    fullWidth={fullWidth}
                    noLabelBorder={noLabelBorder}
                  />
                }
              />
            ))
        ) : (
          <LabelDropdown
            disabled={disabled}
            projectId={projectId}
            value={value}
            onChange={onChange}
            hideDropdownArrow={hideDropdownArrow}
            buttonClassName={buttonClassName}
            fullWidth={fullWidth}
            fullHeight={fullHeight}
            label={
              <LabelSummary
                isMobile={isMobile}
                fullWidth={fullWidth}
                noLabelBorder={noLabelBorder}
                disabled={disabled}
                projectLabels={projectLabels}
                value={value}
              />
            }
          />
        )
      ) : (
        <LabelDropdown
          disabled={disabled}
          projectId={projectId}
          value={value}
          onChange={onChange}
          hideDropdownArrow={hideDropdownArrow}
          buttonClassName={buttonClassName}
          fullWidth={fullWidth}
          fullHeight={fullHeight}
          label={
            <NoLabel
              isMobile={isMobile}
              noLabelBorder={noLabelBorder}
              fullWidth={fullWidth}
              placeholderText={placeholderText}
            />
          }
        />
      )}
    </>
  );
});
