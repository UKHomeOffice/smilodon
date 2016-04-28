package main

import (
	"log"
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
