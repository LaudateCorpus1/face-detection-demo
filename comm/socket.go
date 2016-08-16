package comm

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/ubuntu/face-detection-demo/messages"
)

var socketpath string

const socketfilename string = "facedetect.socket"

// SetSocketDir to initialize socket dir and path between client and server
func SetSocketDir(socketdir string) {
	socketpath = path.Join(socketdir, socketfilename)
}

// StartSocketListener executes a socket listener in its own goroutine
func StartSocketListener(actions chan<- *messages.Action, shutdown <-chan interface{}, wg *sync.WaitGroup) {

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer os.Remove(socketpath)

		l, err := net.Listen("unix", socketpath)
		if err != nil {
			log.Fatal("listen error:", err)
		}

		go func() {
			for {
				conn, err := l.Accept()
				if err != nil {
					select {
					default:
						fmt.Println("Error accepting connection: ", err)
						continue
					case <-shutdown:
						// channel is closed as listener not a real error, we are quitting
						return
					}
				}
				go fetchSocketMessage(conn, actions)
			}
		}()

		<-shutdown
		// this causes l.Accept() to return and exit the coroutine
		l.Close()

	}()

}

// SendToSocket will send an action message to socket message
func SendToSocket(msg *messages.Action) (err error) {
	conn, err := net.Dial("unix", socketpath)
	if err != nil {
		fmt.Println("Couldn connect to socket:", err)
		return
	}
	defer conn.Close()

	data, err := proto.Marshal(msg)
	if err != nil {
		fmt.Println("Can't convert received data to protobuf message:", err)
	}

	if _, err = conn.Write(data); err != nil {
		fmt.Println("Couldn't write to socket:", err)
		return
	}

	return nil
}

func fetchSocketMessage(conn net.Conn, actions chan<- *messages.Action) {
	defer conn.Close()

	msg := new(messages.Action)

	result := make([]byte, 0, 4096)
	length := 0
	buf := make([]byte, 256)
	for {
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Println("Error receiving data: ", err)
			return
		}
		result = append(result, buf[:n]...)
		length += n
		if err == io.EOF {
			break
		}
	}

	if err := proto.Unmarshal(result[:length], msg); err != nil {
		fmt.Println("Receiving not well formatted data: ", err, "as:", result[:length])
	}

	actions <- msg
}
