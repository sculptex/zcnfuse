// zcnfs implements a fuse file system layer to a 0chain storage.
// By sculptex
// Based on https://github.com/bazil/fuse
// for Linux
// TODO
//   Cross-platform compatibility checks (Linux only tested so far)
//   Improve cache (age/delete)
//   Improve error handling
//   Compile stats
//   Calculate costs
//   Uploads
//   Partial downloads
// OTHER
//   Use GoSDK directly instead of CLI tools
// NOTES
//   On error, mount may persist but be broken, use https://github.com/bazil/fuse/cmd/fuse-abort to remove broken fuse mounts
// v0.0.2
//   Improved parameters
//   Inherit user permissions of parent (mountpoint folder)
// v0.0.1
//   Initial Release

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"

	"io"
	"io/ioutil"
	"time"
    "os/exec"
    "os/user"
	"encoding/json"
	"strings"
	"strconv"
     	
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

const version = "0.0.2"

type File struct {
	fs *FS
	path    string
	name    string
	kind    string
	size   uint64
}

type Dir struct {
	fs *FS
	path    string
	name    string
	kind    string
	size   uint64
}

var Filez []File
var Dirz []Dir

const defaultconfig = "config.yaml"
const defaultwallet = "wallet.json"
const defaultallocationfile = "allocation.txt"
const defaultmountpoint = "zcnfuse"

var zcnpath string
var cachepath string
var allocation string

var mountpoint string
var configfile string
var allocationfile string
var walletfile string
var debug string

var fileuid uint32
var filegid uint32

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s [allocation]:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s MOUNTPOINT\n", os.Args[0])
	flag.PrintDefaults()
}


func microTime() float64 {
	loc, _ := time.LoadLocation("UTC")
	now := time.Now().In(loc)
	micSeconds := float64(now.Nanosecond()) / 1000000000
	return float64(now.Unix()) + micSeconds
}

type remotefile struct{
	uid	uint64
	name string
	path string
	kind string
}


type filedata struct{
	name string
	path string
	size uint64
	kind string
	mime string
}

var remotefiles []remotefile
var filez []filedata
var maxrf uint64

func getremotefileuidbypath(path string) int64 {
	for _, rf := range remotefiles {
		if(rf.path == path) {
			return(int64(rf.uid))
		}
	}
	return(-1)
}

func getremotefilepathbyuid(uid uint64) string {
	for _, rf := range remotefiles {
		if(rf.uid == uid) {
			return(rf.path)
		}
	}
	return("")
}

func addremotefile(path string, name string, kind string, size uint64) int64 {
	var rf remotefile
	var fz File
	var uid int64
	rf.path = path
	rf.name = name
	rf.uid = maxrf
	rf.kind = kind
	
	fz.path = path
	fz.name = name
	fz.size = size
	fz.kind = kind
	uid = getremotefileuidbypath(path)
	if(uid<0) {
		Filez = append(Filez, fz)
		remotefiles = append(remotefiles, rf)
		uid = int64(maxrf)
		maxrf++
		fmt.Printf("Added file %d, %s\n", uid, rf.path)
	}
	return(uid)
}

func addremotedir(path string, name string, kind string, size uint64) int64 {
	var rf remotefile
	var dz Dir
	var uid int64
	rf.path = path
	rf.name = name
	rf.uid = maxrf
	rf.kind = kind
	
	dz.path = path
	dz.name = name
	dz.size = size
	dz.kind = kind
	uid = getremotefileuidbypath(path)
	if(uid<0) {
		Dirz = append(Dirz, dz)
		remotefiles = append(remotefiles, rf)
		uid = int64(maxrf)
		maxrf++
		fmt.Printf("Added dir %d, %s\n", uid, rf.path)
	}
	return(uid)
}

func WriteToFile(filename string, data string) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = io.WriteString(file, data)
    if err != nil {
        return err
    }
    return file.Sync()
}

func listfiles(path string) []filedata {
	fmt.Println("LISTFILES "+path)

	var jsonres []byte
	safepath := strings.Replace(path, "/", "_", -1)
	var rescachefile = cachepath+"/"+allocation+safepath+"_res.json"

	if _, err := os.Stat(rescachefile); err == nil {
		//fmt.Printf("File exists\n");  
	    jsonres, err = ioutil.ReadFile(rescachefile)
	    if err != nil {
	        log.Fatal(err)
	    }		
		fmt.Println("Loaded from cache")
	} else {
		var cmdarray []string
		remotepath := path
	    	
		cmdarray = []string{
				zcnpath+"/zbox",
				"list",
				"--json",
				"--allocation",
				string(allocation),
				"--remotepath",
				string(remotepath) }		

		if(configfile != defaultconfig) {
			cmdarray = append( cmdarray, "--config", configfile )	
		}
		
		if(walletfile != defaultwallet) {
			cmdarray = append( cmdarray, "--wallet", walletfile )	
		}	
		
		fmt.Printf("ZBOX LIST %s\n", remotepath)				
						
		head := cmdarray[0]
		parts := cmdarray[1:len(cmdarray)]
		
		//var starttime float64
		//var endtime float64
		//var elapsedtime float64
	
		//starttime = microTime()
		//_ , err = exec.Command(head,parts...).Output()
		//endtime = microTime()
		//elapsedtime = endtime-starttime
			
		jsonres , err := exec.Command(head,parts...).Output()
	
		if err != nil {
			log.Fatal(err)
		}
		WriteToFile(rescachefile, string(jsonres))
		fmt.Println("Loaded, saved to cache")

	}
  	

	var filedat filedata
	var filedats []filedata

	var results []map[string]interface{}
	json.Unmarshal([]byte(jsonres), &results)

	for _, result := range results {

		
		filedat.path = result["path"].(string)
		filedat.name = result["name"].(string)
		filedat.size = uint64(int(result["actual_size"].(float64)))
		filedat.kind = result["type"].(string)
		if(filedat.kind == "f") {
			filedat.mime = result["mimetype"].(string)
		}
				
		filedats = append(filedats, filedat)
	}
	return(filedats)
}



type FS struct{}

func (FS) Root() (fs.Node, error) {
	return &Dirz[0], nil
	//return &Dir{}, nil
}

func (d *Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0o751
	//a.Mode = os.ModeDir | 0o555
	//a.Mode = os.ModeDir | 0o777
	a.Uid = fileuid
	a.Gid = filegid
	return nil
}

func (d *Dir) Lookup(ctx context.Context, path string) (fs.Node, error) {
	var i = 0
	
	var lookuppath string
	
	if(d.path == "/") {
		lookuppath = d.path+path
	} else {
		lookuppath = d.path+"/"+path
	}
	
	for _, f := range Filez {
		if(f.path == lookuppath) {
			fmt.Printf("LOOKUPDIR %s file\n", lookuppath)
			return &Filez[i] , nil
		}		
		i++
	}
	i = 0
	for _, d := range Dirz {
		if(d.path == lookuppath) {
			fmt.Printf("LOOKUPDIR %s dir\n", lookuppath)
			return &Dirz[i] , nil
		}		
		i++
	}	
	fmt.Printf("LOOKUPDIR %s FAIL\n", lookuppath)
	return nil, syscall.ENOENT
}

func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var mydirDirs []fuse.Dirent
	var fe fuse.Dirent
	var files []filedata

	fmt.Printf("READDIRALL %s\n", string(d.path))
	
	files = listfiles(d.path)
	for _ , fd := range files {
		fe.Inode = 0
		fe.Name = fd.name
		if(fd.kind == "f") {			
			fe.Type = fuse.DT_File
		} else {
			fe.Type = fuse.DT_Dir
		}

		mydirDirs = append(mydirDirs , fe)
		
		if(getremotefileuidbypath(fd.path)<1) {
			if(fd.kind == "f") {			
				//fmt.Printf("Adding file %s\n", fe.Name)
				addremotefile(fd.path, fd.name, fd.kind, fd.size)
			} else {
				//fmt.Printf("Adding dir %s\n", fe.Name)
				addremotedir(fd.path, fd.name, fd.kind, fd.size)
			}						

		}
	}
	return mydirDirs, nil
}






func (f *File) Attr(ctx context.Context, a *fuse.Attr) error {
	if(f.kind == "f") {
		a.Mode = 0o644
		//a.Mode = 0o444
		//a.Mode = 0o777
		a.Uid = fileuid
		a.Gid = filegid
		a.Size = uint64(f.size)
	}
	if(f.kind == "d") {
		a.Mode = os.ModeDir | 0o751
		//a.Mode = os.ModeDir | 0o555
		//a.Mode = os.ModeDir | 0o777
		a.Uid = fileuid
		a.Gid = filegid
		a.Size = 0
	}
	return nil
}

func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
		
	var cmdarray []string
	remotepath := f.path // "/"
	
	safepath := strings.Replace(remotepath, "/", "_", -1)
	safepath = strings.Replace(safepath, " ", "_", -1)

	localpath := cachepath+"/"+allocation+safepath //+"_"+f.name
	
	fmt.Printf("READALL %s\n", string(f.path))

	if _, err := os.Stat(localpath); err == nil {
		// LOAD FROM CACHE
	} else {
	
		cmdarray = []string{
				zcnpath+"/zbox",
				"download",
				"--allocation",
				string(allocation),
				"--remotepath",
				string(remotepath),
				"--localpath",
				string(localpath) }		
				
		if(configfile != defaultconfig) {
			cmdarray = append( cmdarray, "--config", configfile )	
		}
		
		if(walletfile != defaultwallet) {
			cmdarray = append( cmdarray, "--wallet", walletfile )	
		}					
		
		fmt.Printf("ZBOX DOWNLOAD %s\n", remotepath)
						
		head := cmdarray[0]
		parts := cmdarray[1:len(cmdarray)]
		
		_ , err = exec.Command(head,parts...).Output()
			
		if err != nil {
			log.Fatal(err)
		}
	}
	
	resfile , err := ioutil.ReadFile(localpath)
	
	if err != nil {
		return nil, syscall.ENOENT
	}
	return resfile, nil
}


func main() {

	fmt.Printf("VERSION %s\n", version)
	
    // Allow user to specify config file    
    flag.StringVar(&mountpoint, "mountpoint", string(defaultmountpoint), "")
	
    // Allow user to specify config file    
    flag.StringVar(&configfile, "config", string(defaultconfig), "")

    // Allow user to specify wallet file    
    flag.StringVar(&walletfile, "wallet", string(defaultwallet), "")
    
    // Allow user to specify allocation file    
    flag.StringVar(&allocation, "allocation", string(""), "allocation (default contents of allocation.txt)")

    flag.Parse()


	//var fileinfo os.FileInfo
	

	if mountpoint == defaultmountpoint {
		if _, err := os.Stat(mountpoint); err == nil {
			fmt.Printf("MOUNTPOINT %s exists\n", string(mountpoint))
		} else {
			err := os.Mkdir(mountpoint, 0755)
			//err := os.Mkdir(mountpoint, 0777)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("MOUNTPOINT %s created\n", string(mountpoint))
		}
	}
	
	fileinfo , err := os.Stat(mountpoint)
	if err != nil {	
		filesys := fileinfo.Sys()
		uid := fmt.Sprint(filesys.(*syscall.Stat_t).Uid)
		fileuid64 , _ := strconv.ParseInt(uid, 10, 32)
		fileuid = uint32(fileuid64)			
		gid := fmt.Sprint(filesys.(*syscall.Stat_t).Gid)
		filegid64 , _ := strconv.ParseInt(gid, 10, 32)	
		filegid = uint32(filegid64)			
	}	

	cachepath = "zcncache"	
	if _, err := os.Stat(cachepath) ; err != nil {
		err := os.Mkdir(cachepath, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}

	maxrf = 0
	
	user, err := user.Current()
    if err != nil {
        log.Fatalf(err.Error())
    }	
	zcnpath = user.HomeDir+"/.zcn"
	
	fmt.Printf("PATH %s\n", string(zcnpath))
	
	
	if len(allocation) != 64 {
	    all, err := ioutil.ReadFile(zcnpath+"/allocation.txt")
	    if err != nil {
	        os.Exit(3)
	    }
	    allocation = string(all)
	}	
    
	addremotedir("/", "/" , "d", 0)
	    	
	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("zcn"),
		fuse.Subtype("zcnfs"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()


	err = fs.Serve(c, FS{})
	if err != nil {
		log.Fatal(err)
	}
}

