/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { AddOutline, ChevronDownOutline, InfoOutline } from "@makeplane/propel/icons";
import { Disclosure, Transition } from "@headlessui/react";
// nerve imports
import type { ModuleLink } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useModules } from "@/hooks/store/use-modules";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { ModuleLinkModal } from "../links/create-update-modal";
import { ModuleLinksList } from "../links/list";

type Props = {
  moduleId: string;
  /** Whether the caller adds, changes and removes the module's links (the sidebar's root decides it). */
  editable: boolean;
};

/**
 * The links of a module's progress sidebar (P6a spec 2.8, R10, R11), once the store has them: every caller who reads
 * the module sees them, a guest too (none yet, it says so); the caller who writes the module (editable: canWriteModules,
 * the module not archived) adds and changes them in the link's dialog, which is this section's, and removes them at
 * once. A link goes once nerve has removed it, and the section says so; refused, it stays, and the section says why.
 * Its follow-ups are those of the session and the sidebar it was asked from (followInSession): once the sidebar has
 * gone, nerve's late answer reaches the store alone.
 */
export const ModuleSidebarLinks = observer(function ModuleSidebarLinks(props: Props) {
  const { moduleId, editable } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getModuleLinks, removeLink } = useModules();
  const toastRefusal = useRefusalToast();
  // states
  const [dialog, setDialog] = useState<{ isOpen: boolean; link?: ModuleLink }>({ isOpen: false });
  const [removing, setRemoving] = useState<ReadonlySet<string>>(new Set());
  const out = useRef(new Set<string>());
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const links = getModuleLinks(moduleId);

  const remove = async (link: ModuleLink) => {
    if (out.current.has(link.id)) return;
    out.current.add(link.id);
    setRemoving(new Set(out.current));
    await followInSession(() => removeLink(link.id), {
      done: () => {
        if (!mounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("links.toasts.removed.title"),
          message: t("links.toasts.removed.message"),
        });
      },
      failed: (error) => {
        if (mounted.current) toastRefusal(error);
      },
    });
    out.current.delete(link.id);
    if (mounted.current) setRemoving(new Set(out.current));
  };

  // the store has yet to read them: the section opens on what nerve gives
  if (links === undefined) return null;

  const addLink = editable && (
    <button
      type="button"
      className="flex items-center gap-1.5 text-13 font-medium text-accent-primary"
      onClick={() => setDialog({ isOpen: true })}
    >
      <AddOutline className="h-3 w-3" />
      {t("common.add_link")}
    </button>
  );

  return (
    <div className="flex flex-col">
      <ModuleLinkModal
        isOpen={dialog.isOpen}
        handleClose={() => setDialog((shown) => ({ ...shown, isOpen: false }))}
        moduleId={moduleId}
        link={dialog.link}
      />

      <div className="flex w-full flex-col items-center justify-start gap-2 border-t border-subtle px-1.5 py-5">
        <Disclosure defaultOpen={links.length > 0}>
          {({ open }) => (
            <div className={`relative flex h-full w-full flex-col ${open ? "" : "flex-row"}`}>
              <Disclosure.Button className="flex w-full items-center justify-between gap-2 p-1.5">
                <span className="text-13 font-medium text-secondary">{t("common.links")}</span>
                <ChevronDownOutline className={`h-3.5 w-3.5 ${open ? "rotate-180 transform" : ""}`} aria-hidden />
              </Disclosure.Button>
              <Transition as="div" show={open}>
                <Disclosure.Panel>
                  <div className="mt-2 flex min-h-72 w-full flex-col space-y-3 overflow-y-auto">
                    {links.length > 0 ? (
                      <>
                        {addLink && <div className="flex w-full items-center justify-end">{addLink}</div>}
                        <ModuleLinksList
                          moduleId={moduleId}
                          editable={editable}
                          removing={removing}
                          onEdit={(link) => setDialog({ isOpen: true, link })}
                          onRemove={(link) => void remove(link)}
                        />
                      </>
                    ) : (
                      <div className="flex items-center justify-between gap-2">
                        <div className="flex items-center gap-2">
                          <InfoOutline className="h-3.5 w-3.5 stroke-[1.5] text-tertiary" />
                          <span className="p-0.5 text-11 text-tertiary">{t("common.no_links_added_yet")}</span>
                        </div>
                        {addLink}
                      </div>
                    )}
                  </div>
                </Disclosure.Panel>
              </Transition>
            </div>
          )}
        </Disclosure>
      </div>
    </div>
  );
});
