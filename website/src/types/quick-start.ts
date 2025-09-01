import { LucideIcon } from "lucide-react";

export interface IDEConfig {
  id: string;
  name: string;
  icon: LucideIcon;
  local?: {
    description: string;
    code: string;
    note?: string;
  };
  remote?: {
    description: string;
    code?: string;
    alternativeCode?: string;
    alternativeDescription?: string;
    note?: string;
    command?: string;
  };
}

export interface InstallationStep {
  step: number;
  title: string;
  description: string;
  code?: string;
  codeLanguage?: string;
  benefits?: string[];
  hasIDETabs?: boolean;
}

export type InstallationType = "local" | "remote";