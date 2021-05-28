# zcnfuse
Fuse library implementation for 0Chain Storage 

## zcnfs implements a fuse file system layer to a 0chain storage.
* By sculptex
* for Linux
* Based on hello example by Bazil.org

# Pre-requisits

Note those are for development purposes, to be able to read/write.

## Install zboxcli

Follow the doc [install zboxcli](https://github.com/0chain/zboxcli/wiki/Build-Linux)

## Create a wallet 

Follow the doc [create a wallet](https://github.com/0chain/zboxcli#Register)

Note you need to uncomment the prefered_blobbers section

## Install zwalletcli

Follow the doc [install zwalletcli](https://github.com/0chain/zwalletcli#1-installation)

## Get some ZCN from the faucet

Follow the doc [get some zcn](https://github.com/0chain/zwalletcli#2-run-zwallet-commands)

## Create a new allocation

Follow the doc [create a new allocation](https://github.com/0chain/zboxcli#create-new-allocation)


# Build

go build .


## Example Use
* zcnfuse -mountpoint test
* zcnfuse -help
```
-allocation string
    allocation (default contents of allocation.txt)
-config string
    (default "config.yaml")
-mountpoint string
    (default "zcnfuse")
-wallet string
    (default "wallet.json")
```
## RELEASES
### v0.0.2
* Improved Command line paramaters
* Inherit mountpoint folder ownership for fuse mount
### v0.0.1
* Initial Release
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
