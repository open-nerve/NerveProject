/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
// nerve imports
import { ChevronRightOutline } from "@makeplane/propel/icons";
import type { Module } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
// local imports
import { ModuleStatusDropdown } from "../module-status-dropdown";

type Props = {
  module: Module;
  /** Whether the caller changes the module here (the sidebar's root decides it). */
  editable: boolean;
  handleClose: () => void;
};

/**
 * The head of a module's progress sidebar (P6a spec 2.6): its close button, named; the module's status, which the
 * caller changes there while editable (ModuleStatusDropdown); its name and its description, plain text.
 */
export const ModuleSidebarHeader = observer(function ModuleSidebarHeader(props: Props) {
  const { module, editable, handleClose } = props;
  // hooks
  const { t } = useTranslation();

  return (
    <>
      <div className="sticky top-0 z-10 flex items-center justify-between bg-surface-1 pt-5">
        <button
          type="button"
          aria-label={t("close")}
          className="flex size-5 items-center justify-center rounded-full bg-layer-3 hover:bg-layer-3-hover"
          onClick={handleClose}
        >
          <ChevronRightOutline className="size-3 text-on-color" aria-hidden="true" />
        </button>
      </div>
      <div className="flex flex-col gap-3">
        <div className="flex items-center gap-5 pt-2">
          <ModuleStatusDropdown moduleId={module.id} editable={editable} />
        </div>
        <h4 className="w-full text-18 font-semibold wrap-break-word text-primary">{module.name}</h4>
        {module.description && (
          <p className="text-14 break-words whitespace-pre-wrap text-secondary">{module.description}</p>
        )}
      </div>
    </>
  );
});
