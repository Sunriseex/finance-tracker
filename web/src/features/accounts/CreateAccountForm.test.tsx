import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { CreateAccountForm } from "./CreateAccountForm";

const mocks = vi.hoisted(() => ({
  createAccount: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  api: {
    createAccount: mocks.createAccount,
    createTransaction: vi.fn(),
    createInterestRule: vi.fn(),
  },
}));

describe("CreateAccountForm", () => {
  it("creates an account through the API client", async () => {
    const user = userEvent.setup();
    const onDone = vi.fn();
    mocks.createAccount.mockResolvedValueOnce({ id: "account-1" });

    render(
      <QueryClientProvider client={new QueryClient()}>
        <CreateAccountForm onDone={onDone} />
      </QueryClientProvider>,
    );

    await user.type(screen.getByLabelText("Name"), "Daily card");
    await user.type(screen.getByLabelText("Bank"), "Test Bank");
    await user.click(screen.getByRole("button", { name: "Create" }));

    await waitFor(() => expect(mocks.createAccount).toHaveBeenCalledWith(expect.objectContaining({
      name: "Daily card",
      bank: "Test Bank",
      type: "card",
      currency: "RUB",
    })));
    expect(onDone).toHaveBeenCalled();
  });
});
