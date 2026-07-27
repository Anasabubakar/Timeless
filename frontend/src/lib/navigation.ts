import {
  LayoutDashboard,
  Building2,
  Target,
  Users,
  Mail,
  BarChart3,
  Settings,
  Bot,
  Zap,
  FolderKanban,
  Activity,
  FileText,
  Link2,
  Upload,
  type LucideIcon,
} from "lucide-react";

export interface NavItem {
  name: string;
  href: string;
  icon: LucideIcon;
  group: string;
  /** Shown in the mobile bottom dock. Keep this to the 4 most-used destinations. */
  dock?: boolean;
}

// Single source of truth for navigation, shared by the desktop sidebar, the
// command palette, and the mobile nav dialog/dock so they never drift apart.
export const NAVIGATION: NavItem[] = [
  { name: "Dashboard", href: "/dashboard", icon: LayoutDashboard, group: "Workspace", dock: true },
  { name: "Activity", href: "/activities", icon: Activity, group: "Workspace" },
  { name: "Import", href: "/import", icon: Upload, group: "Workspace" },

  { name: "Campaigns", href: "/campaigns", icon: Target, group: "CRM" },
  { name: "Sponsors", href: "/sponsors", icon: FolderKanban, group: "CRM", dock: true },
  { name: "Proposals", href: "/proposals", icon: FileText, group: "CRM" },
  { name: "Companies", href: "/companies", icon: Building2, group: "CRM" },
  { name: "Contacts", href: "/contacts", icon: Users, group: "CRM" },
  { name: "Outreach", href: "/outreach", icon: Mail, group: "CRM" },

  { name: "AI Agents", href: "/ai-agents", icon: Bot, group: "AI", dock: true },
  { name: "Automations", href: "/automations", icon: Zap, group: "AI" },

  { name: "Analytics", href: "/analytics", icon: BarChart3, group: "Analytics", dock: true },

  { name: "Integrations", href: "/integrations", icon: Link2, group: "Integrations" },

  { name: "Settings", href: "/settings", icon: Settings, group: "Settings" },
];

export const NAV_GROUP_ORDER = ["Workspace", "CRM", "AI", "Analytics", "Integrations", "Settings"];

export const DOCK_ITEMS = NAVIGATION.filter((item) => item.dock);

export function groupedNavigation(): { group: string; items: NavItem[] }[] {
  return NAV_GROUP_ORDER.map((group) => ({
    group,
    items: NAVIGATION.filter((item) => item.group === group),
  })).filter((g) => g.items.length > 0);
}
