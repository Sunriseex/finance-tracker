import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { Button, Dialog, Input } from ".";

function DialogHarness() {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onClick={() => setOpen(true)}>Open dialog</Button>
      {open ? (
        <Dialog title="Edit account" onClose={() => setOpen(false)}>
          <label>
            Name
            <Input defaultValue="Savings" />
          </label>
          <Button>Save</Button>
        </Dialog>
      ) : null}
    </>
  );
}

describe("Dialog", () => {
  it("sets dialog semantics, closes on Escape, and restores focus", async () => {
    const user = userEvent.setup();
    render(<DialogHarness />);

    const opener = screen.getByRole("button", { name: "Open dialog" });
    await user.click(opener);

    const dialog = screen.getByRole("dialog", { name: "Edit account" });
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(screen.getByRole("button", { name: "Close dialog" })).toHaveFocus();

    await user.keyboard("{Escape}");

    expect(screen.queryByRole("dialog", { name: "Edit account" })).not.toBeInTheDocument();
    expect(opener).toHaveFocus();
  });

  it("keeps tab focus inside the dialog", async () => {
    const user = userEvent.setup();
    render(<DialogHarness />);

    await user.click(screen.getByRole("button", { name: "Open dialog" }));
    await user.tab({ shift: true });

    expect(screen.getByRole("button", { name: "Save" })).toHaveFocus();
  });
});
