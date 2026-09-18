# ABC-101: Export invoices as PDF

Customers ask support for PDF copies every month. See REQ-7~1: it already
covers authentication, and this sentence declares nothing.

## Acceptance criteria

- **ABC-101.AC1~1**: WHEN a customer clicks "Export" THE SYSTEM SHALL
  download the invoice as a PDF
  named after the invoice number.

  The file name matters to accountants.
- **ABC-101.AC2:** WHEN the invoice has more than 50 lines THE SYSTEM SHALL paginate the PDF.
* `ABC-101.AC3~4`: THE PDF SHALL look professional.
  - Non-testable: judged by the brand team at the monthly review.
1. ABC-101.AC4~2: THE export SHALL feel instant.
   - **Non-testable:**
- [ ] ABC-101.AC5~x: WHEN offline THE SYSTEM SHALL queue the export.
- ABC-101.AC6~1:

```markdown
- **ABC-101.AC9~1**: WHEN quoted in a code block THE SYSTEM SHALL ignore me.
```

### REQ-8~3: Audit trail ###

WHEN an invoice is exported THE SYSTEM SHALL record who exported it.

Auditors read this log quarterly.

#### Notes

- **REQ-9~1**: THE SYSTEM SHALL keep exports for 30 days.
