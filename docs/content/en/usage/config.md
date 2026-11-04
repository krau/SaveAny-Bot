---
title: "File Naming & Conflict Strategies"
weight: 11
---

# File Naming & Conflict Strategies

SaveAny-Bot lets you customize how saved files are named and how collisions with existing files are resolved, directly in Telegram with `/config` and `/template`.

## `/config` — User Settings

The `/config` command contains the following per-user settings:

- **Filename strategy** — how the saved file is named
- **Duplicate file strategy** — what happens when a file with the same name already exists in the target storage
- **Silent mode** — save incoming files directly to the default location
- **Default storage** — choose the storage (and directory) used by silent mode and unmatched rules
- **Filename template** — view the current template and the `/template` usage

Settings are stored per user and apply to all of that user's subsequent save/transfer tasks.

### Filename strategy

| Option | Behavior |
|---|---|
| `Default` | Use the original media filename, or a generated name when no original filename is available |
| `Gen From Msg First` | Generate the filename from the message content (e.g. caption, text) and prefer that over the original filename |
| `Template` | Render the filename from a custom template you define with `/template` |

### Duplicate file strategy

| Option | Behavior |
|---|---|
| `Always rename` (default) | Keep the existing file and save the new one with an alternate name |
| `Ask every time` | Prompt you with inline buttons each time a collision occurs |
| `Always overwrite` | Replace the existing file with the new one |
| `Always skip` | Do nothing for conflicting files |

{{< hint info >}}
The conflict strategy only kicks in for storage backends that can detect the existence of a file. Backends that do not support existence checks will fall back to overwriting.
{{< /hint >}}

## `/template` — Custom Filename Template

When the filename strategy is set to `Template`, SaveAny-Bot renders each saved file's name using the template configured via `/template`.

```
/template [template]
```

- Running `/template` without arguments shows your current template and the help text.
- Running it with a template string sets that template as your filename template.

The template uses Go [`text/template`](https://pkg.go.dev/text/template) syntax. The available variables are:

| Variable | Description |
|---|---|
| `{{.msgid}}` | Telegram message ID |
| `{{.msgtags}}` | Hashtags found in the message, joined with `_` |
| `{{.msggen}}` | Filename generated from the message |
| `{{.msgdate}}` | Message date, formatted `YYYY-MM-DD_HH-MM-SS` |
| `{{.msgraw}}` | Raw, unprocessed message text |
| `{{.origname}}` | The media's original filename (if any) |
| `{{.chatid}}` | Chat ID of the message |

Examples:

```
# Fixed prefix + message id + date
/template Image_{{.msgid}}_{{.msgdate}}.jpg

# Use original name if available, otherwise a generated name
/template {{.origname}}
```

{{< hint info >}}
When an album is saved as a batch (media groups sent to the bot, and watched albums matched by a `NEW-FOR-ALBUM` rule), files without their own caption use the album's caption for template variables and message rules, so `{{.msgtags}}` and `{{.msgraw}}` are populated for every item. If the template produces the same name for several items, later items keep their generated name.
{{< /hint >}}

{{< hint warning >}}
The template only takes effect when the filename strategy is set to `Template`. If template parsing fails, SaveAny-Bot falls back to the default filename naming logic.
{{< /hint >}}