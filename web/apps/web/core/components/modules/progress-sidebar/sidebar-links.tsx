/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { AddOutline, ChevronDownOutline, InfoOutline } from "@makeplane/propel/icons";
import { Disclosure, Transition } from "@headlessui/react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { ILinkDetails, ModuleLink } from "@nerve/types";
// hooks
import { useModule } from "@/hooks/store/use-module";
// local imports
import { CreateUpdateModuleLinkModal, ModuleLinksList } from "../links";

type Props = {
  moduleId: string;
  workspaceSlug: string;
  projectId: string;
  /** Whether the caller adds, changes and removes the module's links (the sidebar's root decides it). */
  editable: boolean;
};

/** The links of a module's progress sidebar, as the module store before nerve's had them. */
export const ModuleSidebarLinks = observer(function ModuleSidebarLinks(props: Props) {
  const { moduleId, workspaceSlug, projectId, editable } = props;
  // states
  const [moduleLinkModal, setModuleLinkModal] = useState(false);
  const [selectedLinkToUpdate, setSelectedLinkToUpdate] = useState<ILinkDetails | null>(null);
  // store hooks
  const { t } = useTranslation();
  const { getModuleById, createModuleLink, updateModuleLink, deleteModuleLink } = useModule();
  const links = getModuleById(moduleId)?.link_module ?? [];

  const handleCreateLink = async (formData: ModuleLink) => {
    const payload = { metadata: {}, ...formData };
    await createModuleLink(workspaceSlug, projectId, moduleId, payload);
  };

  const handleUpdateLink = async (formData: ModuleLink, linkId: string) => {
    const payload = { metadata: {}, ...formData };
    await updateModuleLink(workspaceSlug, projectId, moduleId, linkId, payload);
  };

  const handleDeleteLink = async (linkId: string) => {
    try {
      await deleteModuleLink(workspaceSlug, projectId, moduleId, linkId);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "Module link deleted successfully.",
      });
    } catch (_error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Some error occurred",
      });
    }
  };

  const handleEditLink = (link: ILinkDetails) => {
    setSelectedLinkToUpdate(link);
    setModuleLinkModal(true);
  };

  const addLink = editable && (
    <button
      className="flex items-center gap-1.5 text-13 font-medium text-accent-primary"
      onClick={() => setModuleLinkModal(true)}
    >
      <AddOutline className="h-3 w-3" />
      {t("add_link")}
    </button>
  );

  return (
    <div className="flex flex-col">
      <CreateUpdateModuleLinkModal
        isOpen={moduleLinkModal}
        handleClose={() => {
          setModuleLinkModal(false);
          setTimeout(() => {
            setSelectedLinkToUpdate(null);
          }, 500);
        }}
        data={selectedLinkToUpdate}
        createLink={handleCreateLink}
        updateLink={handleUpdateLink}
      />
      <div className="flex w-full flex-col items-center justify-start gap-2 border-t border-subtle px-1.5 py-5">
        <Disclosure defaultOpen={links.length > 0}>
          {({ open }) => (
            <div className={`relative flex h-full w-full flex-col ${open ? "" : "flex-row"}`}>
              <Disclosure.Button className="flex w-full items-center justify-between gap-2 p-1.5">
                <div className="flex items-center justify-start gap-2 text-13">
                  <span className="font-medium text-secondary">{t("common.links")}</span>
                </div>
                <div className="flex items-center gap-2.5">
                  <ChevronDownOutline
                    className={`h-3.5 w-3.5 ${open ? "rotate-180 transform" : ""}`}
                    aria-hidden="true"
                  />
                </div>
              </Disclosure.Button>
              <Transition as="div" show={open}>
                <Disclosure.Panel>
                  <div className="mt-2 flex min-h-72 w-full flex-col space-y-3 overflow-y-auto">
                    {links.length > 0 ? (
                      <>
                        {addLink && <div className="flex w-full items-center justify-end">{addLink}</div>}
                        <ModuleLinksList
                          moduleId={moduleId}
                          handleEditLink={handleEditLink}
                          handleDeleteLink={handleDeleteLink}
                          disabled={!editable}
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
