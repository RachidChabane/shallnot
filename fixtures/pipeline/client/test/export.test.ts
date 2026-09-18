import { describe, expect, it } from "vitest";
import { renderInvoicePage } from "../src/invoice-page";

describe("invoice page", () => {
  it("starts the download in place when Export is clicked [verifies ABC-101.AC2~1]", async () => {
    const page = renderInvoicePage("INV-3");
    await page.click("Export");
    expect(page.location).toBe("/invoices/INV-3");
    expect(page.downloads).toEqual(["INV-3.pdf"]);
  });
});
