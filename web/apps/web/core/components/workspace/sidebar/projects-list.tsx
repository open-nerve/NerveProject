/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState, useRef, useEffect } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";
import { observer } from "mobx-react";
import { matchPath, useParams, useLocation } from "react-router";
import { AddOutline, ChevronRightOutline, MoreHorizontalOutline } from "@makeplane/propel/icons";
import { Transition } from "@headlessui/react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { IconButton, getIconButtonStyling } from "@nerve/propel/icon-button";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { Loader } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { CreateProjectModal } from "@/components/project/create-project-modal";
import { SidebarNavItem } from "@/components/sidebar/sidebar-navigation";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { useCopyProjectLink } from "@/hooks/use-copy-link";
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";
// local imports
import { SidebarProjectsListItem } from "./projects-list-item";
import { useProjectDrop } from "./use-project-drop";

export const SidebarProjectsList = observer(function SidebarProjectsList() {
  // states
  const [isAllProjectsListOpen, setIsAllProjectsListOpen] = useState(true);
  const [isProjectModalOpen, setIsProjectModalOpen] = useState(false);
  const [isScrolled, setIsScrolled] = useState(false); // scroll animation state
  // refs
  const containerRef = useRef<HTMLDivElement | null>(null);
  // store hooks
  const { t } = useTranslation();
  const { toggleCreateProjectModal } = useCommandPalette();
  const { canCreateProject } = useUserPermissions();
  const { preferences: projectPreferences } = useProjectNavigationPreferences();
  const { isExtendedProjectSidebarOpened, toggleExtendedProjectSidebar } = useAppTheme();

  const { loader, joinedProjectIds: joinedProjects } = useProject();
  const handleOnProjectDrop = useProjectDrop();
  // router params
  const { workspaceSlug } = useParams();
  const { pathname } = useLocation();

  // auth
  const isAuthorizedUser = canCreateProject(workspaceSlug);

  // Compute limited projects for main sidebar
  const displayedProjects = projectPreferences.showLimitedProjects
    ? joinedProjects.slice(0, projectPreferences.limitedProjectsCount)
    : joinedProjects;

  // Check if there are more projects to show
  const hasMoreProjects =
    projectPreferences.showLimitedProjects && joinedProjects.length > projectPreferences.limitedProjectsCount;

  const copyProjectLink = useCopyProjectLink();

  /**
   * Implementing scroll animation styles based on the scroll length of the container
   */
  useEffect(() => {
    const handleScroll = () => {
      if (containerRef.current) {
        const scrollTop = containerRef.current.scrollTop;
        setIsScrolled(scrollTop > 0);
      }
    };
    const currentContainerRef = containerRef.current;
    if (currentContainerRef) {
      currentContainerRef.addEventListener("scroll", handleScroll);
    }
    return () => {
      if (currentContainerRef) {
        currentContainerRef.removeEventListener("scroll", handleScroll);
      }
    };
  }, [containerRef]);

  useEffect(() => {
    const element = containerRef.current;

    if (!element) return;

    return combine(
      autoScrollForElements({
        element,
        canScroll: ({ source }) => source?.data?.dragInstanceId === "PROJECTS",
        getAllowedAxis: () => "vertical",
      })
    );
  }, [containerRef]);

  const toggleListDisclosure = (isOpen: boolean) => {
    setIsAllProjectsListOpen(isOpen);
    localStorage.setItem("isAllProjectsListOpen", isOpen.toString());
  };
  useEffect(() => {
    if (matchPath({ path: "/:workspaceSlug/projects", end: false }, pathname)) {
      setIsAllProjectsListOpen(true);
      localStorage.setItem("isAllProjectsListOpen", "true");
    }
  }, [pathname]);
  return (
    <>
      {workspaceSlug && (
        <CreateProjectModal
          isOpen={isProjectModalOpen}
          onClose={() => setIsProjectModalOpen(false)}
          workspaceSlug={workspaceSlug}
        />
      )}
      <div
        ref={containerRef}
        className={cn({
          "border-t border-strong": isScrolled,
        })}
      >
        <>
          <div className="flex flex-col">
            <div className="group flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-placeholder hover:bg-layer-transparent-hover">
              {/* the list's heading, named by its words, says whether the list shows; isAllProjectsListOpen alone
              decides that, as the projects' pages open the list too (above) */}
              <button
                type="button"
                className="flex w-full items-center gap-1 text-left text-13 font-semibold whitespace-nowrap text-placeholder"
                onClick={() => toggleListDisclosure(!isAllProjectsListOpen)}
                aria-expanded={isAllProjectsListOpen}
              >
                <span className="text-13 font-semibold">{t("projects")}</span>
              </button>
              <div className="flex items-center gap-1">
                {isAuthorizedUser && (
                  <Tooltip label={t("create_project")}>
                    <IconButton
                      variant="ghost"
                      size="sm"
                      icon={AddOutline}
                      onClick={() => {
                        setIsProjectModalOpen(true);
                      }}
                      className="text-placeholder opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                      aria-label={t("aria_labels.projects_sidebar.create_new_project")}
                    />
                  </Tooltip>
                )}
                {/* the heading opens and closes the list as this does: this one, the pointer's alone, is neither a Tab
                stop nor read */}
                <button
                  type="button"
                  onClick={() => toggleListDisclosure(!isAllProjectsListOpen)}
                  className={cn(getIconButtonStyling("ghost", "sm"), "text-placeholder")}
                  tabIndex={-1}
                  aria-hidden="true"
                >
                  <ChevronRightOutline
                    className={cn("size-3.5 transition-transform", {
                      "rotate-90": isAllProjectsListOpen,
                    })}
                  />
                </button>
              </div>
            </div>
            <Transition
              as="div"
              show={isAllProjectsListOpen}
              enter="transition duration-100 ease-out"
              enterFrom="transform scale-95 opacity-0"
              enterTo="transform scale-100 opacity-100"
              leave="transition duration-75 ease-out"
              leaveFrom="transform scale-100 opacity-100"
              leaveTo="transform scale-95 opacity-0"
            >
              {loader === "init-loader" && (
                <Loader className="w-full space-y-1.5">
                  {["first", "second", "third", "fourth"].map((row) => (
                    <Loader.Item key={row} height="28px" />
                  ))}
                </Loader>
              )}
              {workspaceSlug && isAllProjectsListOpen && (
                <div className="flex flex-col gap-0.5">
                  <>
                    {displayedProjects.map((projectId, index) => (
                      <SidebarProjectsListItem
                        key={projectId}
                        workspaceSlug={workspaceSlug}
                        projectId={projectId}
                        handleCopyText={() => void copyProjectLink(projectId)}
                        projectListType="JOINED"
                        isLastChild={index === displayedProjects.length - 1}
                        handleOnProjectDrop={handleOnProjectDrop}
                      />
                    ))}
                    {hasMoreProjects && (
                      <SidebarNavItem>
                        <button
                          type="button"
                          onClick={() => toggleExtendedProjectSidebar()}
                          className="flex flex-grow items-center gap-1.5 text-13 font-medium text-tertiary"
                          id="extended-project-sidebar-toggle"
                        >
                          <MoreHorizontalOutline className="size-4 flex-shrink-0" />
                          <span>
                            {t(isExtendedProjectSidebarOpened ? "sidebar.fewer_projects" : "sidebar.more_projects")}
                          </span>
                        </button>
                      </SidebarNavItem>
                    )}
                  </>
                </div>
              )}
            </Transition>
          </div>
        </>

        {isAuthorizedUser && joinedProjects?.length === 0 && (
          <button
            type="button"
            className="flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-13 leading-5 font-medium text-secondary hover:bg-surface-2"
            onClick={() => {
              toggleCreateProjectModal(true);
            }}
          >
            {t("add_project")}
          </button>
        )}
      </div>
    </>
  );
});
