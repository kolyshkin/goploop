package ploop

// #cgo CFLAGS: -D_GNU_SOURCE
// #cgo LDFLAGS: -lploop -lxml2 -lrt
// #include <ploop/libploop.h>
import "C"

// Possible SetVerboseLevel arguments
const (
	NoConsole  = C.PLOOP_LOG_NOCONSOLE
	NoStdout   = C.PLOOP_LOG_NOSTDOUT
	Timestamps = C.PLOOP_LOG_TIMESTAMPS
)

// SetVerboseLevel sets a level of verbosity when logging to stdout/stderr
func SetVerboseLevel(v int) {
	C.ploop_set_verbose_level(C.int(v))
}

// SetLogFile enables logging to a file and sets log file name
func SetLogFile(file string) error {
	cfile := C.CString(file)
	defer cfree(cfile)

	ret := C.ploop_set_log_file(cfile)

	return mkerr(ret)
}

// SetLogLevel sets a level of verbosity when logging to a file
func SetLogLevel(v int) {
	C.ploop_set_log_level(C.int(v))
}

// Ploop is a type containing DiskDescriptor.xml opened by the library
type Ploop struct {
	d *C.struct_ploop_disk_images_data
}

// Open opens a ploop DiskDescriptor.xml, most ploop operations require it
func Open(file string) (Ploop, error) {
	var d Ploop
	cfile := C.CString(file)
	defer cfree(cfile)

	ret := C.ploop_open_dd(&d.d, cfile)

	return d, mkerr(ret)
}

// Close closes a ploop disk descriptor when it is no longer needed
func Close(d Ploop) {
	C.ploop_close_dd(d.d)
}

type ImageMode int

// Possible values for ImageMode
const (
	Expanded     ImageMode = C.PLOOP_EXPANDED_MODE
	Preallocated ImageMode = C.PLOOP_EXPANDED_PREALLOCATED_MODE
	Raw          ImageMode = C.PLOOP_RAW_MODE
)

// CreateParam is a set of parameters for a newly created ploop
type CreateParam struct {
	Size uint64 // image size, in kilobytes (FS size is about 10% smaller)
	Mode ImageMode
	File string // path to and a file name for base delta image
}

// Create creates a ploop image and its DiskDescriptor.xml
func Create(p *CreateParam) error {
	var a C.struct_ploop_create_param

	// default image file name
	if p.File == "" {
		p.File = "root.hdd"
	}

	a.size = convertSize(p.Size)
	a.mode = C.int(p.Mode)
	a.image = C.CString(p.File)
	defer cfree(a.image)
	a.fstype = C.CString("ext4")
	defer cfree(a.fstype)

	ret := C.ploop_create_image(&a)
	return mkerr(ret)
}

// MountParam is a set of parameters to pass to Mount()
type MountParam struct {
	Uuid     string // snapshot uuid (empty for top delta)
	Target   string // mount point (empty if no mount is needed)
	Flags    int    // bit mount flags such as MS_NOATIME
	Data     string // auxiliary mount options
	Readonly bool   // mount read-only
	Fsck     bool   // do fsck before mounting inner FS
	Quota    bool   // enable quota for inner FS
}

// Mount creates a ploop device and (optionally) mounts it
func Mount(d Ploop, p *MountParam) (string, error) {
	var a C.struct_ploop_mount_param
	var device string

	if p.Uuid != "" {
		a.guid = C.CString(p.Uuid)
		defer cfree(a.guid)
	}
	if p.Target != "" {
		a.target = C.CString(p.Target)
		defer cfree(a.target)
	}

	// mount_data should not be NULL
	a.mount_data = C.CString(p.Data)
	defer cfree(a.mount_data)

	a.flags = C.int(p.Flags)
	a.ro = bool2cint(p.Readonly)
	a.fsck = bool2cint(p.Fsck)
	a.quota = bool2cint(p.Quota)

	ret := C.ploop_mount_image(d.d, &a)
	if ret == 0 {
		device = C.GoString(&a.device[0])
		// TODO? fsck_code = C.GoString(a.fsck_rc)
	}
	return device, mkerr(ret)
}

// Umount unmounts the ploop filesystem and dismantles the device
func Umount(d Ploop) error {
	ret := C.ploop_umount_image(d.d)

	return mkerr(ret)
}

// Resize changes the ploop size. Online resize is recommended.
func Resize(d Ploop, size uint64, offline bool) error {
	var p C.struct_ploop_resize_param

	p.size = convertSize(size)
	p.offline_resize = bool2cint(offline)

	ret := C.ploop_resize_image(d.d, &p)
	return mkerr(ret)
}

// Snapshot creates a ploop snapshot, returning its uuid
func Snapshot(d Ploop) (string, error) {
	var p C.struct_ploop_snapshot_param
	var uuid, err = Uuid()
	if err != nil {
		return "", err
	}
	p.guid = C.CString(uuid)
	defer cfree(p.guid)

	ret := C.ploop_create_snapshot(d.d, &p)
	if ret == 0 {
		uuid = C.GoString(p.guid)
	}

	return uuid, mkerr(ret)
}

// SwitchSnapshot makes a specified snapshot a top one, losing the old one.
func SwitchSnapshot(d Ploop, uuid string) error {
	var p C.struct_ploop_snapshot_switch_param

	p.guid = C.CString(uuid)
	defer cfree(p.guid)

	ret := C.ploop_switch_snapshot_ex(d.d, &p)

	return mkerr(ret)
}

// DeleteSnapshot deletes a snapshot (merging it down if necessary)
func DeleteSnapshot(d Ploop, uuid string) error {
	cuuid := C.CString(uuid)
	defer cfree(cuuid)

	ret := C.ploop_delete_snapshot(d.d, cuuid)

	return mkerr(ret)
}

type ReplaceFlag int

// Possible values for ReplaceParam.flags
const (
	// KeepName renames the new file to old file name after replace;
	// note that if this option is used the old file is removed.
	KeepName ReplaceFlag = C.PLOOP_REPLACE_KEEP_NAME
)

// ReplaceParam is a set of parameters to Replace()
type ReplaceParam struct {
	File string // new image file name
	// Image to be replaced is specified by either
	// uuid, current file name, or level,
	// in the above order of preference.
	Uuid    string
	CurFile string
	Level   int
	Flags   ReplaceFlag
}

// Replace replaces a ploop image to a different (but identical) one
func Replace(d Ploop, p *ReplaceParam) error {
	var a C.struct_ploop_replace_param

	a.file = C.CString(p.File)
	defer cfree(a.file)

	if p.Uuid != "" {
		a.guid = C.CString(p.Uuid)
		defer cfree(a.guid)
	} else if p.CurFile != "" {
		a.cur_file = C.CString(p.CurFile)
		defer cfree(a.cur_file)
	} else {
		a.level = C.int(p.Level)
	}

	a.flags = C.int(p.Flags)

	ret := C.ploop_replace_image(d.d, &a)

	return mkerr(ret)
}

// FSInfoData holds information about ploop inner file system
type FSInfoData struct {
	Blocksize   uint64
	Blocks      uint64
	Blocks_free uint64
	Inodes      uint64
	Inodes_free uint64
}

// FSInfo gets info of ploop's inner file system
func FSInfo(file string) (FSInfoData, error) {
	var cinfo C.struct_ploop_info
	var info FSInfoData
	cfile := C.CString(file)
	defer cfree(cfile)

	ret := C.ploop_get_info_by_descr(cfile, &cinfo)
	if ret == 0 {
		info.Blocksize = uint64(cinfo.fs_bsize)
		info.Blocks = uint64(cinfo.fs_blocks)
		info.Blocks_free = uint64(cinfo.fs_bfree)
		info.Inodes = uint64(cinfo.fs_inodes)
		info.Inodes_free = uint64(cinfo.fs_ifree)
	}

	return info, mkerr(ret)
}

// ImageInfoData holds information about ploop image
type ImageInfoData struct {
	Blocks    uint64
	Blocksize uint32
	Version   int
}

// ImageInfo gets information about a ploop image
func ImageInfo(d Ploop) (ImageInfoData, error) {
	var cinfo C.struct_ploop_spec
	var info ImageInfoData

	ret := C.ploop_get_spec(d.d, &cinfo)
	if ret == 0 {
		info.Blocks = uint64(cinfo.size)
		info.Blocksize = uint32(cinfo.blocksize)
		info.Version = int(cinfo.fmt_version)
	}

	return info, mkerr(ret)
}

// Uuid generates a ploop UUID
func Uuid() (string, error) {
	var cuuid [39]C.char

	ret := C.ploop_uuid_generate(&cuuid[0], 39)
	if ret != 0 {
		return "", mkerr(ret)
	}

	uuid := C.GoString(&cuuid[0])
	return uuid, nil
}
