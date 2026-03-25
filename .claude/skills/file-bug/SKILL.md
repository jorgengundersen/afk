---
name: file-bug
description: File a bug found during other work. Either defer or fix inline depending on relatedness.
user-invocable: false
allowed-tools: Bash(bd *)
---

**Trigger when:** a test fails for a pre-existing reason, or code inspection reveals a defect unrelated to the current change.

```bash
# When working on a claimed task — link it
bd create "<broken thing>" -t bug -p <0-3> \
  -d "<wrong / expected / how found>" \
  --deps discovered-from:<current-task-id> --json

# When no task is claimed — omit --deps
bd create "<broken thing>" -t bug -p <0-3> \
  -d "<wrong / expected / how found>" --json
```

Priority: 0=build/security, 1=major, 2=minor, 3=cosmetic.

**Unrelated to current work:** file it, continue current task.

**Related and straightforward fix:** file it, fix it, update with context, close it, continue.
```bash
bd update <bug-id> --append-notes "<what was fixed and how>" --json
bd close <bug-id> --reason "Fixed inline" --json
```
