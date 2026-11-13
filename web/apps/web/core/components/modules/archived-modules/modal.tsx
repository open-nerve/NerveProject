/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
// nerve imports
import type { Module } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
// hooks
import { useModules } from "@/hooks/store/use-modules";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  /** The module to archive, as it was when the dialog was asked for. */
  module: Module | undefined;
  workspaceSlug: string;
  projectId: string;
};

/**
 * The dialog that archives a completed or cancelled module (P6a spec 2.6, R11), named by its title, in the natural
 * Tab order. It closes only once nerve has archived the module, says so and goes to the modules' list; refused, it
 * stays and says nerve's reason. While the archive is out nothing closes it (Escape, the backdrop, Cancel). Its
 * follow-ups are those of the session it was sent in (followInSession) and of its page: once the page has gone (a Back,
 * another project), nerve's late answer reaches the store alone.
 */
export function ArchiveModuleModal(props: Props) {
  const { isOpen, handleClose, module, workspaceSlug, projectId } = props;
  // router
  const navigate = useNavigate();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { archive } = useModules();
  const toastRefusal = useRefusalToast();
  // states
  const [busy, setBusy] = useState(false);
  const out = useRef(false);
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const submit = async () => {
    if (!module || out.current) return;
    out.current = true;
    setBusy(true);
    await followInSession(() => archive(module.id), {
      done: () => {
        if (!mounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("project_module.action.archive.success.title"),
          message: t("project_module.action.archive.success.description"),
        });
        handleClose();
        return navigate(`/${workspaceSlug}/projects/${projectId}/modules`);
      },
      failed: (error) => {
        if (mounted.current) toastRefusal(error);
      },
    });
    out.current = false;
    if (mounted.current) setBusy(false);
  };

  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={busy ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.LG}
    >
      <div className="px-5 py-4">
        <ModalTitle className="text-18 font-medium 2xl:text-20">
          {t("project_module.archive.title", { name: module?.name })}
        </ModalTitle>
        <p className="mt-3 text-13 text-secondary">{t("project_module.archive.body")}</p>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={handleClose} disabled={busy}>
            {t("cancel")}
          </Button>
          <Button variant="primary" size="lg" onClick={() => void submit()} loading={busy}>
            {busy ? t("common.archiving") : t("archive")}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
