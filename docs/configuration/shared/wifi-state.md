# Wi-Fi State

sing-box can monitor Wi-Fi state to enable routing rules based on `wifi_ssid` and `wifi_bssid`.

## Platform Support

| Platform        | Support          | Notes                    |
|-----------------|------------------|--------------------------|
| Android         | :material-check: | In graphical client      |
| Apple platforms | :material-check: | In graphical clients     |
| Linux           | :material-check: | Requires supported daemon |
| Windows         | :material-check: | WLAN API                 |
| Others          | :material-close: |                          |

## Linux

The following backends are supported and will be auto-detected in order of priority:

| Backend          | Interface   |
|------------------|-------------|
| NetworkManager   | D-Bus       |
| IWD              | D-Bus       |
| wpa_supplicant   | Unix socket |
| ConnMan          | D-Bus       |

## Windows

Uses Windows WLAN API.
