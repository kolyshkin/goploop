package ploop

// A test suite, also serving as an example of how to use the package

import (
	"github.com/dustin/go-humanize"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

var (
	old_pwd  string
	test_dir string
	d        Ploop
	snap     string
)

const baseDelta = "root.hdd"

func TestPrepare(t *testing.T) {
	var err error

	old_pwd, err = os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %s", err)
	}

	test_dir, err = ioutil.TempDir(old_pwd, "tmp-test")
	if err != nil {
		t.Fatalf("TempDir %q: %s", old_pwd, err)
	}

	err = os.Chdir(test_dir)
	if err != nil {
		t.Fatalf("Chdir %q: %s", test_dir, err)
	}

	SetVerboseLevel(255)
	SetLogLevel(1)
	SetLogFile("ploop.log")

	t.Logf("Running tests in %s", test_dir)
}

func TestUuid(t *testing.T) {
	uuid, e := Uuid()
	if e != nil {
		t.Errorf("Uuid: %s", e)
	}

	t.Logf("Got uuid %s", uuid)
}

func TestCreate(t *testing.T) {
	size := "512M"
	var p CreateParam

	s, e := humanize.ParseBytes(size)
	if e != nil {
		t.Fatalf("humanize.ParseBytes: can't parse %s: %s", size, e)
	}
	p.size = s
	p.file = baseDelta

	e = Create(&p)
	if e != nil {
		t.Fatalf("Create: %s", e)
	}
}

func TestOpen(t *testing.T) {
	var e error

	d, e = Open("DiskDescriptor.xml")
	if e != nil {
		t.Errorf("Open: %s ", e)
	}
}

func TestMount(t *testing.T) {
	mnt := "mnt"

	e := os.Mkdir(mnt, 0755)
	if e != nil {
		t.Fatalf("os.Mkdir: %s", e)
	}

	p := MountParam{target: mnt}
	dev, e := Mount(d, &p)
	if e != nil {
		t.Fatalf("Open: %s", e)
	} else {
		t.Logf("Mounted; ploop device %s", dev)
	}
}

func TestOnlineDownsize(t *testing.T) {
	size := "384M"
	s, e := humanize.ParseBytes(size)
	if e != nil {
		t.Fatalf("humanize.ParseBytes: can't parse %s: %s", size, e)
	}

	e = Resize(d, s, false)
	if e != nil {
		t.Fatalf("Resize to %s (%d) failed: %s", size, s, e)
	}
}

func TestSnapshot(t *testing.T) {
	uuid, e := Snapshot(d)
	if e != nil {
		t.Fatalf("Snapshot: %s", e)
	} else {
		t.Logf("Created online Snapshot; uuid %s", uuid)
	}

	snap = uuid
}

func copyFile(src, dst string) error {

	cmd := exec.Command("cp", "-a", src, dst)
	err := cmd.Run()

	return err
}

func testReplace(t *testing.T) {
	var p ReplaceParam
	newDelta := baseDelta + ".new"
	e := copyFile(baseDelta, newDelta)
	if e != nil {
		t.Fatalf("copyFile: %s", e)
	}

	p.file = newDelta
	p.curFile = baseDelta
	p.flags = keepName
	e = Replace(d, &p)
	if e != nil {
		t.Fatalf("Replace: %s", e)
	}
}

func TestReplaceOnline(t *testing.T) {
	testReplace(t)
}

func TestSnapshotDelete(t *testing.T) {
	e := DeleteSnapshot(d, snap)
	if e != nil {
		t.Fatalf("DeleteSnapshot: %s", e)
	} else {
		t.Logf("Deleted snapshot %s", snap)
	}
}

func TestUmount(t *testing.T) {
	e := Umount(d)
	if e != nil {
		t.Fatalf("Umount: %s", e)
	}
}

func TestSnapshotOffline(t *testing.T) {
	uuid, e := Snapshot(d)
	if e != nil {
		t.Fatalf("Snapshot: %s", e)
	} else {
		t.Logf("Created offline Snapshot; uuid %s", uuid)
	}

	snap = uuid
}

func TestReplaceOffline(t *testing.T) {
	testReplace(t)
}

func TestSnapshotSwitch(t *testing.T) {
	e := SwitchSnapshot(d, snap)
	if e != nil {
		t.Fatalf("SwitchSnapshot: %s", e)
	} else {
		t.Logf("Switched to snapshot %s", snap)
	}
}

func TestFSInfo(t *testing.T) {
	i, e := FSInfo("DiskDescriptor.xml")

	if e != nil {
		t.Errorf("FSInfo: %v", e)
	} else {
		bTotal := i.blocks * i.blocksize
		bAvail := i.blocks_free * i.blocksize
		bUsed := bTotal - bAvail

		iTotal := i.inodes
		iAvail := i.inodes_free
		iUsed := iTotal - iAvail

		t.Logf("\n             Size       Used      Avail Use%%\n%7s %9s %10s %10s %3d%%\n%7s %9d %10d %10d %3d%%",
			"Blocks",
			humanize.Bytes(bTotal),
			humanize.Bytes(bUsed),
			humanize.Bytes(bAvail),
			100*bUsed/bTotal,
			"Inodes",
			iTotal,
			iUsed,
			iAvail,
			100*iUsed/iTotal)
		t.Logf("\nInode ratio: 1 inode per %s of disk space",
			humanize.Bytes(bTotal/iTotal))
	}
}

func TestImageInfo(t *testing.T) {
	i, e := ImageInfo(d)
	if e != nil {
		t.Errorf("ImageInfo: %v", e)
	} else {
		t.Logf("\n              Blocks  Blocksize       Size  Ver\n%20d %10d %10s %4d",
			i.blocks, i.blocksize,
			humanize.Bytes(512*i.blocks),
			i.version)
	}

}

// TestCleanup is the last test, removing files created by previous tests
func TestCleanup(t *testing.T) {
	Close(d)
	os.Chdir(old_pwd)
	os.RemoveAll(test_dir)
}
