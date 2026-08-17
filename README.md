# call-detect

A small Windows background program that notices when an app is using the **microphone** or **webcam**, shows that in the notification area, and can POST a JSON webhook when the state changes.

It does not talk to Discord, Zoom, or any other app by name. It reads Windows privacy records, live audio capture sessions, and camera streaming activity, so it works with current and future programs — including a browser preview that uses the webcam with no microphone.

## What it reports

| Field | Meaning |
|-------|---------|
| `busy` | Microphone **or** webcam is in use |
| `microphone` | Microphone is in use |
| `webcam` | Webcam is in use |
| `sources` | Short names of apps that currently hold a device |
| `updated_at` | When that snapshot was published |

Speaker or headset **playback** (music, videos) does not count. Game voice chat does, because the microphone has an active capture session.

State is debounced for about two seconds so a brief device grab does not flicker the tray icon or the webhook.

## Privacy

call-detect only **reads** Windows ConsentStore timestamps, audio session metadata, and camera sensor-activity process names. It does not open, record, or stream the microphone or camera. There is no telemetry. The webhook is optional and off by default.

## Install

Download the latest `call-detect.exe`:

**[call-detect.exe](https://github.com/scallister/call-detect/releases/latest/download/call-detect.exe)**

That link always follows the newest [release](https://github.com/scallister/call-detect/releases/latest). Or [build it](#build). Then from a terminal:

```text
call-detect.exe --install
```

That copies the program to `%LOCALAPPDATA%\call-detect\`, writes a sample `config.yaml` if you do not already have one, starts it now, and runs it again at logon for the current user. No administrator rights.

If call-detect is already installed, running a newer downloaded `call-detect.exe` (double-click or from a terminal, without flags) asks whether to replace the installed copy and restart it. `--install` does the same replacement without asking.

```text
call-detect.exe --uninstall
```

Removes the logon entry. Files stay in `%LOCALAPPDATA%\call-detect\` until you delete them. Use **Quit** on the tray icon to stop a running copy. An update will ask the running copy to exit so the file can be replaced.

### Flags

| Flag | Purpose |
|------|---------|
| `--install` | Install and start at logon |
| `--uninstall` | Remove logon autostart |
| `--dump` | Print ConsentStore records, live audio sessions, streaming cameras, and the confirmed result |
| `--console` | Also write logs to a console window |
| `--webhook-url` | Override the webhook destination |
| `--config` | Path to `config.yaml` |

Logs are always appended to `%LOCALAPPDATA%\call-detect\call-detect.log`.

## Configuration

Webhook URL, first match wins:

1. `--webhook-url`
2. Environment variable `CALL_DETECT_WEBHOOK_URL`
3. `webhook_url` in `%LOCALAPPDATA%\call-detect\config.yaml`

```yaml
# webhook_url: "http://homeassistant.local:8123/api/webhook/YOUR_WEBHOOK_ID"
```

Leave it unset to run locally (tray + `status.json` only).

The latest snapshot is also written to `%LOCALAPPDATA%\call-detect\status.json`.

## Tray icon

- **Idle:** gray ring, tooltip `call-detect: idle`
- **On a call:** green ring, tooltip such as `call-detect: on a call (mic, Discord.exe)`

Right-click for microphone / webcam / sources, **Install** / **Uninstall** (logon startup), **Set webhook URL...**, **GitHub...** (opens the [project](https://github.com/scallister/call-detect)), and **Quit**. The webhook dialog shows an example JSON payload next to the URL field, writes `config.yaml`, and applies immediately. The icon updates on its own when state changes.

## Webhook

On each debounced change to `busy`, `microphone`, or `webcam`, call-detect POSTs `Content-Type: application/json`. **Set webhook URL...** shows this same example next to the URL field:

```json
{
  "busy": true,
  "microphone": true,
  "webcam": false,
  "sources": ["Discord.exe"],
  "updated_at": "2026-08-17T12:00:00Z"
}
```

Failed deliveries are retried a few times. 4xx responses (except 429) are not retried.

## Home Assistant

You can drive a helper or a light from that JSON. One simple setup:

1. **Settings → Devices & services → Helpers** — create a Toggle, for example `On a call` (`input_boolean.on_a_call`).
2. **Settings → Automations & scenes → Create automation** — add a **Webhook** trigger.
3. Allow **POST**. Enable **Only accessible from the local network** (`local_only`).
4. Copy the webhook ID. The URL is:

   `http://<home-assistant-host>:8123/api/webhook/<webhook_id>`

   Use `https://` if that is how you reach Home Assistant.
5. Set that URL as `webhook_url` (or `--webhook-url` / `CALL_DETECT_WEBHOOK_URL`).
6. In the automation actions, turn the helper on or off from `trigger.json.busy`.

Example:

```yaml
alias: Call detect webhook
trigger:
  - platform: webhook
    allowed_methods:
      - POST
    local_only: true
    webhook_id: YOUR_WEBHOOK_ID
action:
  - choose:
      - conditions:
          - condition: template
            value_template: "{{ trigger.json.busy }}"
        sequence:
          - action: input_boolean.turn_on
            target:
              entity_id: input_boolean.on_a_call
    default:
      - action: input_boolean.turn_off
        target:
          entity_id: input_boolean.on_a_call
```

A second automation can turn a light (or anything else) on when `input_boolean.on_a_call` is on.

## Build

Current stable Go. Cross-compile from Linux or macOS:

```text
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-H windowsgui -s -w -X github.com/scallister/call-detect/internal/version.Version=v0.0.0" -o call-detect.exe ./cmd/call-detect
```

`-H windowsgui` hides the console when the program is started from Explorer. Use `--console` or `--dump` when you want a terminal.

```text
go test ./...
```

## How detection works

**Microphone.** Windows stores last-used times under:

`HKCU\Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\microphone`

Packaged apps have their own subkeys. Desktop apps are under `NonPackaged`. Those keys stay “in use” (`LastUsedTimeStop` is zero) after many apps leave a call. call-detect treats the microphone as in use only when ConsentStore says so **and** that same app has an active WASAPI **capture** session. Playback-only apps (music, videos) never match.

**Webcam.** A video-only page (for example a browser camera test) often has no audio session at all. Webcam state comes from the Windows camera sensor activity monitor: processes that are actually streaming a camera. Idle Discord leftover ConsentStore records do not count.

If the camera monitor cannot start, webcam falls back to ConsentStore plus a capture or render session. If both live sources fail, call-detect uses ConsentStore alone.

The program must run as the signed-in user (not as a Windows service under Local System) so it can read that user’s keys.
