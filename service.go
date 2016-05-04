package main

import (
	"github.com/coreos/go-systemd/dbus"
	"log"
)

type srv struct {
	name            string
	state           string
	needsRestarting bool
	dbusConn        *dbus.Conn
}

func (s *srv) restart() (err error) {
	// RestartUnit allows to start a stopped service
	_, err = s.dbusConn.RestartUnit(s.name, "", nil)
	if err != nil {
		log.Printf("Error starting unit %q: %q.\n", s.name, err)
		return err
	}
	log.Printf("Service %q start scheduled. See 'journalctl -u %s' for more details.\n", s.name, s.name)
	return nil
}

// func (s *srv) status() (err error) {
// 	us, err := s.dbusConn.ListUnits()
// 	for _, u := range us {
// 		if u.Name ==
// 	}

// 	return nil

// }
