package ports

import (
	"fmt"
	"net"
	"time"
)

// Waits until it is possible to connect to the given address using TCP.
func WaitForConnect(locator string, timeout time.Duration) error {
	start := time.Now()

	var conn net.Conn
	var err error
	for time.Now().Sub(start) < timeout {
		conn, err = net.DialTimeout("tcp", locator, timeout)
		if err == nil {
			break
		}
	}
	if conn != nil {
		conn.Close()
	}
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", locator, err)
	}
	return nil
}
