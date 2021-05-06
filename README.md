# zcnfuse
Fuse library implementation for 0Chain Storage 

## zcnfs implements a fuse file system layer to a 0chain storage.
* By sculptex
* Based on https://github.com/bazil/fuse
* for Linux
## TODO
* Cross-platform compatibility checks (Linux only tested so far)
* Improve cache (age/delete)
* Improve error handling
* Compile stats
* Calculate costs
* Uploads
* Partial downloads
## OTHER
* Use GoSDK directly instead of CLI tools
## NOTES
* On error, mount may persist but be broken, use https://github.com/bazil/fuse/cmd/fuse-abort to remove broken fuse mounts
