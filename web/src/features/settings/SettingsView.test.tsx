import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Profile } from "../../api/types";
import { SettingsView } from "./SettingsView";

const mocks = vi.hoisted(() => ({
  updateProfile: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  ApiClientError: class ApiClientError extends Error {},
  api: {
    updateProfile: mocks.updateProfile,
  },
}));

const profile: Profile = {
  user: {
    id: "user-1",
    email: "user@example.com",
    primary_currency: "RUB",
  },
};

function renderSettingsView(inputProfile?: Profile) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

  render(
    <QueryClientProvider client={queryClient}>
      <SettingsView profile={inputProfile} />
    </QueryClientProvider>,
  );

  return { invalidateQueries };
}

describe("SettingsView", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    mocks.updateProfile.mockReset();
  });

  it("disables currency and save controls without profile", () => {
    renderSettingsView();

    expect(screen.getByLabelText("Email")).toHaveValue("");
    expect(screen.getByLabelText("Primary currency")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save settings" })).toBeDisabled();
  });

  it("updates primary currency, invalidates dependent queries, and shows Saved", async () => {
    const user = userEvent.setup();
    mocks.updateProfile.mockResolvedValue({
      user: {
        ...profile.user,
        primary_currency: "USD",
      },
    });
    const { invalidateQueries } = renderSettingsView(profile);

    await user.selectOptions(screen.getByLabelText("Primary currency"), "USD");
    await user.click(screen.getByRole("button", { name: "Save settings" }));

    await waitFor(() => {
      expect(mocks.updateProfile).toHaveBeenCalledWith({ primary_currency: "USD" });
      expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["profile"] });
      expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["dashboard"] });
    });
    expect(await screen.findByText("Saved")).toBeInTheDocument();
  });

  it("shows API errors and does not show Saved", async () => {
    const user = userEvent.setup();
    mocks.updateProfile.mockRejectedValue(new Error("Profile update failed"));
    renderSettingsView(profile);

    await user.selectOptions(screen.getByLabelText("Primary currency"), "USD");
    await user.click(screen.getByRole("button", { name: "Save settings" }));

    expect(await screen.findByText("Profile update failed")).toBeInTheDocument();
    expect(screen.queryByText("Saved")).not.toBeInTheDocument();
  });
});
