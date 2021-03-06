package agent

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/chrislusf/netchan/agent/store"
	"github.com/chrislusf/netchan/util"
)

func (as *AgentServer) handleLocalReadConnection(conn net.Conn, name string, offset int64) {
	as.name2StoreLock.Lock()
	ds, ok := as.name2Store[name]
	if !ok {
		s, err := store.NewLocalFileDataStore(as.dir, fmt.Sprintf("%s-%d", name, as.Port))
		if err != nil {
			// log.Printf("Failed to create queue on disk: %v", err)
			as.name2StoreLock.Unlock()
			return
		}
		as.name2Store[name] = NewLiveDataStore(s)
		ds = as.name2Store[name]
	}
	as.name2StoreLock.Unlock()

	closeSignal := make(chan bool, 1)

	go func() {
		buf := make([]byte, 4)
		for false {
			// println("wait for reader heartbeat")
			conn.SetReadDeadline(time.Now().Add(2500 * time.Millisecond))
			_, _, err := util.ReadBytes(conn, buf)
			if err != nil {
				fmt.Printf("connection is closed? (%v)\n", err)
				closeSignal <- true
				close(closeSignal)
				return
			}
		}
	}()

	buf := make([]byte, 4)

	// loop for every read
	for {
		_, err := ds.store.ReadAt(buf, offset)
		if err != nil {
			// connection is closed
			if err != io.EOF {
				log.Printf("Read size from %s offset %d: %v", name, offset, err)
			}
			// println("got problem reading", name, offset, err.Error())
			return
		}

		offset += 4
		size := util.BytesToUint32(buf)

		// println("reading", name, offset, "size:", size)

		messageBytes := make([]byte, size)
		_, err = ds.store.ReadAt(messageBytes, offset)
		if err != nil {
			// connection is closed
			if err != io.EOF {
				log.Printf("Read data from %s offset %d: %v", name, offset, err)
			}
			return
		}
		offset += int64(size)

		m := util.LoadMessage(messageBytes)
		// println(name, "sent:", len(messageBytes), ":", string(m.Data()))
		util.WriteBytes(conn, buf, m)

		if m.Flag() != util.Data {
			// println("Finished reading", name)
			break
		}
	}

}
