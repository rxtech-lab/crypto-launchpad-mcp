import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { CreateTokenForm } from "../create-token-form";

describe("CreateTokenForm", () => {
  const mockOnSubmit = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("Form Rendering", () => {
    it("should render all form fields", () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      expect(screen.getByText("Create New JWT Token")).toBeInTheDocument();
      expect(screen.getByLabelText("Token Name *")).toBeInTheDocument();
      expect(screen.getByLabelText("Client ID")).toBeInTheDocument();
      expect(screen.getByLabelText("Expires In")).toBeInTheDocument();
      expect(screen.getByText("Audiences")).toBeInTheDocument();
      expect(screen.getByText("Roles")).toBeInTheDocument();
      expect(screen.getByText("Scopes")).toBeInTheDocument();
    });

    it("should render submit button", () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      const submitButton = screen.getByRole("button", { name: "Create Token" });
      expect(submitButton).toBeInTheDocument();
    });

    it("should show loading state when isLoading is true", () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} isLoading={true} />);

      const submitButton = screen.getByRole("button", { name: "Creating..." });
      expect(submitButton).toBeDisabled();
    });
  });

  describe("Form Validation", () => {
    it("should require token name", async () => {
      // Mock alert
      const alertSpy = vi.spyOn(window, "alert").mockImplementation(() => {});

      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      const submitButton = screen.getByRole("button", { name: "Create Token" });
      fireEvent.click(submitButton);

      expect(alertSpy).toHaveBeenCalledWith("Token name is required");
      expect(mockOnSubmit).not.toHaveBeenCalled();

      alertSpy.mockRestore();
    });

    it("should submit form with valid data", async () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      // Fill in required field
      const tokenNameInput = screen.getByLabelText("Token Name *");
      fireEvent.change(tokenNameInput, { target: { value: "Test Token" } });

      const submitButton = screen.getByRole("button", { name: "Create Token" });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith({
          tokenName: "Test Token",
          aud: [],
          clientId: "",
          roles: [],
          scopes: [],
          expiresIn: "30d",
        });
      });
    });
  });

  describe("Dynamic Fields Management", () => {
    it("should add and remove audiences", async () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      // Add audience
      const audienceInput = screen.getByPlaceholderText(
        "e.g., api.example.com"
      );
      fireEvent.change(audienceInput, { target: { value: "api.test.com" } });

      const addButton = audienceInput.nextElementSibling as HTMLElement;
      fireEvent.click(addButton);

      expect(screen.getByText("api.test.com")).toBeInTheDocument();

      // Remove audience
      const removeButton = screen
        .getByText("api.test.com")
        .parentElement?.querySelector("button");
      if (removeButton) {
        fireEvent.click(removeButton);
      }

      await waitFor(() => {
        expect(screen.queryByText("api.test.com")).not.toBeInTheDocument();
      });
    });

    it("should add audience on Enter key press", () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      const audienceInput = screen.getByPlaceholderText(
        "e.g., api.example.com"
      );
      fireEvent.change(audienceInput, { target: { value: "api.test.com" } });
      fireEvent.keyPress(audienceInput, { key: "Enter", code: "Enter" });

      expect(screen.getByText("api.test.com")).toBeInTheDocument();
    });

    it("should add and remove roles", () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      // Add role
      const roleInput = screen.getByPlaceholderText("e.g., admin, user");
      fireEvent.change(roleInput, { target: { value: "admin" } });

      const addButton = roleInput.nextElementSibling as HTMLElement;
      fireEvent.click(addButton);

      expect(screen.getByText("admin")).toBeInTheDocument();

      // Remove role
      const removeButton = screen
        .getByText("admin")
        .parentElement?.querySelector("button");
      if (removeButton) {
        fireEvent.click(removeButton);
      }

      expect(screen.queryByText("admin")).not.toBeInTheDocument();
    });

    it("should add and remove scopes", () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      // Add scope
      const scopeInput = screen.getByPlaceholderText(
        "e.g., read, write, delete"
      );
      fireEvent.change(scopeInput, { target: { value: "read" } });

      const addButton = scopeInput.nextElementSibling as HTMLElement;
      fireEvent.click(addButton);

      expect(screen.getByText("read")).toBeInTheDocument();

      // Remove scope
      const removeButton = screen
        .getByText("read")
        .parentElement?.querySelector("button");
      if (removeButton) {
        fireEvent.click(removeButton);
      }

      expect(screen.queryByText("read")).not.toBeInTheDocument();
    });
  });

  describe("Form Input Handling", () => {
    it("should update token name input", () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      const tokenNameInput = screen.getByLabelText(
        "Token Name *"
      ) as HTMLInputElement;
      fireEvent.change(tokenNameInput, { target: { value: "My Test Token" } });

      expect(tokenNameInput.value).toBe("My Test Token");
    });

    it("should update client ID input", () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      const clientIdInput = screen.getByLabelText(
        "Client ID"
      ) as HTMLInputElement;
      fireEvent.change(clientIdInput, { target: { value: "test-client-123" } });

      expect(clientIdInput.value).toBe("test-client-123");
    });

    it("should update expiration select", () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      const expirationSelect = screen.getByLabelText(
        "Expires In"
      ) as HTMLSelectElement;
      fireEvent.change(expirationSelect, { target: { value: "7d" } });

      expect(expirationSelect.value).toBe("7d");
    });

    it("should have correct expiration options", () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      const expirationSelect = screen.getByLabelText("Expires In");
      const options = Array.from(expirationSelect.querySelectorAll("option"));

      expect(options).toHaveLength(4);
      expect(options[0]).toHaveTextContent("7 days");
      expect(options[1]).toHaveTextContent("30 days");
      expect(options[2]).toHaveTextContent("90 days");
      expect(options[3]).toHaveTextContent("1 year");
    });
  });

  describe("Form Reset", () => {
    it("should reset form after successful submission", async () => {
      mockOnSubmit.mockResolvedValue(undefined);

      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      // Fill form
      const tokenNameInput = screen.getByLabelText(
        "Token Name *"
      ) as HTMLInputElement;
      fireEvent.change(tokenNameInput, { target: { value: "Test Token" } });

      const clientIdInput = screen.getByLabelText(
        "Client ID"
      ) as HTMLInputElement;
      fireEvent.change(clientIdInput, { target: { value: "test-client" } });

      // Add an audience
      const audienceInput = screen.getByPlaceholderText(
        "e.g., api.example.com"
      );
      fireEvent.change(audienceInput, { target: { value: "api.test.com" } });
      const addButton = audienceInput.nextElementSibling as HTMLElement;
      fireEvent.click(addButton);

      // Submit form
      const submitButton = screen.getByRole("button", { name: "Create Token" });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalled();
      });

      // Check form is reset
      await waitFor(() => {
        expect(tokenNameInput.value).toBe("");
        expect(clientIdInput.value).toBe("");
        expect(screen.queryByText("api.test.com")).not.toBeInTheDocument();
      });
    });
  });

  describe("Complex Form Submission", () => {
    it("should submit form with all fields populated", async () => {
      render(<CreateTokenForm onSubmit={mockOnSubmit} />);

      // Fill basic fields
      fireEvent.change(screen.getByLabelText("Token Name *"), {
        target: { value: "Complex Token" },
      });
      fireEvent.change(screen.getByLabelText("Client ID"), {
        target: { value: "complex-client" },
      });
      fireEvent.change(screen.getByLabelText("Expires In"), {
        target: { value: "7d" },
      });

      // Add audiences
      const audienceInput = screen.getByPlaceholderText(
        "e.g., api.example.com"
      );
      fireEvent.change(audienceInput, { target: { value: "api1.test.com" } });
      fireEvent.click(audienceInput.nextElementSibling as HTMLElement);
      fireEvent.change(audienceInput, { target: { value: "api2.test.com" } });
      fireEvent.click(audienceInput.nextElementSibling as HTMLElement);

      // Add roles
      const roleInput = screen.getByPlaceholderText("e.g., admin, user");
      fireEvent.change(roleInput, { target: { value: "admin" } });
      fireEvent.click(roleInput.nextElementSibling as HTMLElement);
      fireEvent.change(roleInput, { target: { value: "user" } });
      fireEvent.click(roleInput.nextElementSibling as HTMLElement);

      // Add scopes
      const scopeInput = screen.getByPlaceholderText(
        "e.g., read, write, delete"
      );
      fireEvent.change(scopeInput, { target: { value: "read" } });
      fireEvent.click(scopeInput.nextElementSibling as HTMLElement);
      fireEvent.change(scopeInput, { target: { value: "write" } });
      fireEvent.click(scopeInput.nextElementSibling as HTMLElement);

      // Submit form
      const submitButton = screen.getByRole("button", { name: "Create Token" });
      fireEvent.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalledWith({
          tokenName: "Complex Token",
          aud: ["api1.test.com", "api2.test.com"],
          clientId: "complex-client",
          roles: ["admin", "user"],
          scopes: ["read", "write"],
          expiresIn: "7d",
        });
      });
    });
  });
});
