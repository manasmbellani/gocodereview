# GoCmdScanner
Golang script to perform code review across projects by wrapping around `grep` binary.

The project is inspired by the [nuclei](https://github.com/projectdiscovery/nuclei) project.

## Examples

### Standard Usage
To scan for targets which could have SMB Ghost vulnerability in file `smb_smbghost_check.yaml`, the following signature example can be used:
