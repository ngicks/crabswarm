# Status

Current state: not started — plan drafted with automatic decisions
(AD1-AD5), idea gate not confirmed by the user yet.

## Checklist

- [ ] Idea gate confirmed by user (IDEA.md `Gate:` line)
- [ ] Step 1 — proto: `AdminService.Send` + messages (AD2), regenerate
- [ ] Step 2 — extract shared deliver-and-notify helper from
      `service_inbox.go`
- [ ] Step 3 — `AdminService.Send` impl; reserved `admin` sender and
      name rejection (AD3); target grammar reuse + `*` (AD4)
- [ ] Step 4 — cli client `AdminSend` + rendering
- [ ] Step 5 — cobra re-group `chat admin {list,register,move,send}`;
      delete old spellings (AD1); update `chat.go` help text
- [ ] Step 6 — `admin log` hook point for plan 05 (AD5)
- [ ] Step 7 — e2e: admin send attribution, no member row, old
      spellings gone

Next action: user reviews IDEA.md and the automatic decisions; then
step 1.
