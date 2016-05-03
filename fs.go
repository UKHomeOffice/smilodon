package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
)

// hasFs checks if d has a file system created and returns a bool.
func hasFs(d, f string) bool {
	o, err := exec.Command("/usr/bin/lsblk", "-n", "-o", "FSTYPE", d).Output()
	if err != nil {
		log.Printf("Failed to read file system type of %q: %q.\n", d, err)
		// Return true here just to be on the safe side
		// FIXME: I think the process should exit here?
		return true
	}
	fs := strings.Trim(string(o), "\n")
	if fs == f {
		return true
	}
	if fs == "" {
		return false
	}
	log.Printf("Device %q appears to have a %q file system. However specified file system is %q.\n", d, fs, f)
	return true
}

// mkfs creates file system f on device d.
func mkfs(d, f string) error {
	mkfsCmd := "/usr/sbin/mkfs." + f
	cmd := exec.Command(mkfsCmd, "-q", d)
	err := cmd.Run()
	if err != nil {
		log.Printf("Failed to create %q file system on %q device: %q.\n", f, d, err)
		return err
	}
	log.Printf("Successfully formatted device %q with file system %q.\n", d, f)
	return nil
}

// mount mounts device d with file system type t to mount point p and returns an error if any.
func mount(d, p, t string) (err error) {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		log.Printf("Mount point %q does not exist. Creating %q.\n", p, p)
		if err := os.MkdirAll(p, 0750); err != nil {
			log.Printf("Failed to create the mount path: %q.\n", err)
			return err
		}
	}
	log.Printf("Mounting %q to %q.\n", d, p)
	cmd := exec.Command("/usr/bin/mount", "-t", t, d, p)
	o, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Mount failed: %q to %q: %q.\n", d, p, string(o))
		return err
	}
	log.Printf("Successfully mounted device %q to %q.\n", d, p)
	return nil
}

// isMounted checks if device d is mounted. It returns a boolean
func isMounted(d string) bool {
	v, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		log.Printf("Failed to read mounts information from /proc/mounts: %q.\n", err)
	}
	if strings.Contains(string(v), d+" ") {
		return true
	}
	return false
}
