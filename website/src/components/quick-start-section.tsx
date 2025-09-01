"use client";

import React from "react";
import {
  AnimatedContainer,
  StaggerContainer,
} from "@/components/animated-container";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Cloud, Download } from "lucide-react";
import { InstallationStep } from "@/components/ui/installation-step";
import { IDE_CONFIGS } from "@/config/ide-configs";
import { INSTALLATION_STEPS } from "@/config/installation-steps";
import { InstallationType } from "@/types/quick-start";

export function QuickStartSection() {
  // State for dropdown and copy management
  const [selectedIDEs, setSelectedIDEs] = React.useState<
    Record<string, string>
  >({});
  const [dropdownStates, setDropdownStates] = React.useState<
    Record<string, boolean>
  >({});
  const [copyStates, setCopyStates] = React.useState<Record<string, boolean>>(
    {}
  );

  // Initialize selected IDEs when component mounts
  React.useEffect(() => {
    const localIDEs = IDE_CONFIGS.filter((ide) => ide.local);
    const remoteIDEs = IDE_CONFIGS.filter((ide) => ide.remote);

    setSelectedIDEs({
      local: localIDEs[0]?.id || "",
      remote: remoteIDEs[0]?.id || "",
    });
  }, []);

  // Handle click outside for dropdowns
  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const dropdownContainers = document.querySelectorAll(
        '[id^="dropdown-container-"]'
      );
      let shouldClose = false;

      dropdownContainers.forEach((container) => {
        if (!container.contains(event.target as Node)) {
          shouldClose = true;
        }
      });

      if (shouldClose) {
        setDropdownStates({});
      }
    };

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Helper functions for state management
  const handleToggleDropdown = (type: InstallationType) => {
    setDropdownStates((prev) => ({
      ...prev,
      [type]: !prev[type],
    }));
  };

  const handleSelectIDE = (type: InstallationType, ideId: string) => {
    setSelectedIDEs((prev) => ({
      ...prev,
      [type]: ideId,
    }));
    setDropdownStates((prev) => ({
      ...prev,
      [type]: false,
    }));
  };

  const handleCopyStateChange = (codeKey: string, copied: boolean) => {
    setCopyStates((prev) => ({ ...prev, [codeKey]: copied }));
  };

  return (
    <section className="py-20 bg-muted/30">
      <div className="container mx-auto px-4 max-w-4xl">
        <AnimatedContainer className="text-center mb-12">
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">Quick Start</h2>
          <p className="text-lg text-muted-foreground">
            Get up and running in minutes
          </p>
        </AnimatedContainer>

        <Tabs defaultValue="local" className="w-full mb-8">
          <div className="sticky top-0 z-10 pb-2 -mx-4 px-4 py-2">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="local">
                <Download className="mr-1.5 h-3.5 w-3.5" />
                Local Installation
              </TabsTrigger>
              <TabsTrigger value="remote">
                <Cloud className="mr-1.5 h-3.5 w-3.5" />
                Remote MCP Server
              </TabsTrigger>
            </TabsList>
          </div>

          {(["local", "remote"] as const).map((type) => {
            const availableIDEs = IDE_CONFIGS.filter((ide) =>
              type === "local" ? ide.local : ide.remote
            );

            return (
              <TabsContent key={type} value={type}>
                <StaggerContainer className="space-y-8">
                  {INSTALLATION_STEPS[type].map((step) => (
                    <InstallationStep
                      key={step.step}
                      step={step}
                      type={type}
                      availableIDEs={availableIDEs}
                      selectedIDE={
                        selectedIDEs[type] || availableIDEs[0]?.id || ""
                      }
                      dropdownOpen={dropdownStates[type] || false}
                      copyStates={copyStates}
                      onToggleDropdown={() => handleToggleDropdown(type)}
                      onSelectIDE={(ideId) => handleSelectIDE(type, ideId)}
                      onCopyStateChange={handleCopyStateChange}
                    />
                  ))}
                </StaggerContainer>
              </TabsContent>
            );
          })}
        </Tabs>
      </div>
    </section>
  );
}
