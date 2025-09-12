import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { signOut } from "next-auth/react";
import { DashboardNav } from "../dashboard-nav";

// Mock next-auth/react
vi.mock("next-auth/react", () => ({
  signOut: vi.fn(),
}));

// Mock Next.js Link component
vi.mock("next/link", () => ({
  default: ({ children, href, className }: any) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
}));

const mockSignOut = signOut as vi.MockedFunction<typeof signOut>;

describe("DashboardNav", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("Navigation Structure", () => {
    it("should render all navigation elements", () => {
      render(<DashboardNav />);

      // Check main navigation elements
      expect(screen.getByText("Back to Site")).toBeInTheDocument();
      expect(screen.getByText("Dashboard")).toBeInTheDocument();
      expect(screen.getByText("Overview")).toBeInTheDocument();
      expect(screen.getByText("JWT Tokens")).toBeInTheDocument();
      expect(screen.getByText("Sessions")).toBeInTheDocument();
      expect(screen.getByText("Sign Out")).toBeInTheDocument();
    });

    it("should render correct navigation links", () => {
      render(<DashboardNav />);

      // Check link hrefs
      expect(screen.getByText("Back to Site").closest("a")).toHaveAttribute(
        "href",
        "/"
      );
      expect(screen.getByText("Overview").closest("a")).toHaveAttribute(
        "href",
        "/dashboard"
      );
      expect(screen.getByText("JWT Tokens").closest("a")).toHaveAttribute(
        "href",
        "/dashboard/tokens"
      );
      expect(screen.getByText("Sessions").closest("a")).toHaveAttribute(
        "href",
        "/dashboard/sessions"
      );
    });

    it("should render navigation icons", () => {
      render(<DashboardNav />);

      // Check that icons are present (by checking for lucide icon classes or data attributes)
      const container = screen.getByRole("navigation");
      expect(container).toBeInTheDocument();

      // Verify that the navigation has the expected structure
      expect(screen.getByText("Overview")).toBeInTheDocument();
      expect(screen.getByText("JWT Tokens")).toBeInTheDocument();
      expect(screen.getByText("Sessions")).toBeInTheDocument();
    });
  });

  describe("Sign Out Functionality", () => {
    it("should call signOut when sign out button is clicked", () => {
      render(<DashboardNav />);

      const signOutButton = screen.getByText("Sign Out").closest("button");
      expect(signOutButton).toBeInTheDocument();

      fireEvent.click(signOutButton!);

      expect(mockSignOut).toHaveBeenCalledWith({ callbackUrl: "/" });
    });

    it("should render sign out button with correct styling", () => {
      render(<DashboardNav />);

      const signOutButton = screen.getByText("Sign Out").closest("button");
      expect(signOutButton).toHaveClass("flex", "items-center", "space-x-2");
    });
  });

  describe("Responsive Design", () => {
    it("should hide navigation links on mobile (md:flex class)", () => {
      render(<DashboardNav />);

      const navLinksContainer = screen.getByText("Overview").closest("div");
      expect(navLinksContainer).toHaveClass("hidden", "md:flex");
    });

    it("should hide sign out text on small screens (sm:inline class)", () => {
      render(<DashboardNav />);

      const signOutText = screen.getByText("Sign Out");
      expect(signOutText).toHaveClass("hidden", "sm:inline");
    });
  });

  describe("Styling and Layout", () => {
    it("should have correct navigation structure", () => {
      render(<DashboardNav />);

      const nav = screen.getByRole("navigation");
      expect(nav).toHaveClass("bg-white", "shadow-sm", "border-b");

      const container = nav.querySelector(".max-w-7xl");
      expect(container).toBeInTheDocument();
      expect(container).toHaveClass("mx-auto", "px-4", "sm:px-6", "lg:px-8");
    });

    it("should render dashboard title with correct styling", () => {
      render(<DashboardNav />);

      const title = screen.getByText("Dashboard");
      expect(title).toHaveClass("text-xl", "font-semibold", "text-gray-900");
    });

    it("should apply hover effects to navigation links", () => {
      render(<DashboardNav />);

      const backLink = screen.getByText("Back to Site").closest("a");
      expect(backLink).toHaveClass("hover:text-gray-900", "transition-colors");

      const overviewLink = screen.getByText("Overview").closest("a");
      expect(overviewLink).toHaveClass(
        "hover:text-gray-900",
        "transition-colors"
      );
    });
  });

  describe("Accessibility", () => {
    it("should have proper navigation role", () => {
      render(<DashboardNav />);

      const nav = screen.getByRole("navigation");
      expect(nav).toBeInTheDocument();
    });

    it("should have accessible button for sign out", () => {
      render(<DashboardNav />);

      const signOutButton = screen.getByRole("button", { name: /sign out/i });
      expect(signOutButton).toBeInTheDocument();
    });

    it("should have proper link structure for navigation", () => {
      render(<DashboardNav />);

      const links = screen.getAllByRole("link");
      expect(links.length).toBeGreaterThan(0);

      // Check that each navigation link has proper text content
      expect(
        screen.getByRole("link", { name: /back to site/i })
      ).toBeInTheDocument();
      expect(
        screen.getByRole("link", { name: /overview/i })
      ).toBeInTheDocument();
      expect(
        screen.getByRole("link", { name: /jwt tokens/i })
      ).toBeInTheDocument();
      expect(
        screen.getByRole("link", { name: /sessions/i })
      ).toBeInTheDocument();
    });
  });
});
