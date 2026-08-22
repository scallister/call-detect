# Linux

The icon is a [StatusNotifierItem](https://www.freedesktop.org/wiki/Specifications/StatusNotifierItem/). KDE Plasma and many status-notifier hosts show it natively. GNOME needs an AppIndicator / StatusNotifier extension. Dialogs use `zenity` or `kdialog` when present (install one for Install / webhook / self-update prompts). Left-click can also open a zenity list if the host does not show the D-Bus menu.

Right-click the ring for the same menu as [Windows](../README.md#tray-icon) and [macOS](macos.md).

| Idle | On a call |
|------|-----------|
| ![Linux idle](screenshots/linux-idle.png) | ![Linux on a call](screenshots/linux-on-a-call.png) |

![Tray menu when the webhook POST failed](screenshots/linux-webhook-failed.png)

**Set webhook URL...** (zenity):

![Set webhook URL dialog](screenshots/linux-webhook.png)

Same gray / green / red ring: idle, on a call, webhook failed. See the [main README](../README.md#tray-icon).
