## Example config

Refer to the config struct file for available options:
[config/structs.go](config/structs.go)

## Usage

Make sure to set your *arr host and token BEFORE trying to add the download client.
qdebrid mocks the qbittorrent webapi so use `qBittorrent` as your download client option

### Important

Toggle off 'Remove Completed' at the bottom of the download client options

Under 'Settings -> Media Management -> Importing' enable 'Import using scripts' and select the `on_import.sh` script
