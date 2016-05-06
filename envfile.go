package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
)

// writeEnvFile writes an environment file f and returns an error if any. A
// path to a file gets created as well.
func writeEnvFile(f string, i instance) (err error) {
	s := fmt.Sprintf("NODE_IP=%s\nNODE_ID=%s\nVOLUME_ID=%s\nNETWORK_INTERFACE_ID=%s\n",
		i.networkInterface.IPAddress, i.nodeID, i.volume.id, i.networkInterface.id,
	)
	baseDir := path.Dir(f)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		err := os.MkdirAll(baseDir, 0755)
		if err != nil {
			log.Printf("Unable to create environment file path %q: %q.\n", baseDir, err)
		}
	}
	if err := ioutil.WriteFile(f, []byte(s), 0644); err != nil {
		log.Printf("Failed to write an environment file %q: %q.\n", f, err)
		return err
	}
	return nil
}
