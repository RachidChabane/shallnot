# Parking

- **PARK-2~1**: WHEN a stay lasts 15 minutes or less THE SYSTEM SHALL charge 0.00.
  - Example: quote(entry=2026-03-02T10:00, exit=2026-03-02T10:15) => 0.00
  - Example: quote(entry=2026-03-02T10:00, exit=2026-03-02T09:00) => raises InvalidStay
  - Example: quote(entry=

Out of scope (concurrency): one gate per lane.
**Out of scope (payment):** cards only.

- **PARK-3~1**: THE SYSTEM SHALL print a ticket.

  - Example: ticket(lane=3, vip=false, label="A, B => C") => "printed"
