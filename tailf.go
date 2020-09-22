package goo_ws

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"time"
)

type tailF struct {
	filename   string
	conn       *websocket.Conn
	send       chan []byte
	fileOffset int64
}

func TailF(filename string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.IsWebsocket() {
			conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				log.Println(err.Error())
				return
			}
			client := &tailF{
				filename: filename,
				conn:     conn,
				send:     make(chan []byte, 256),
			}
			go client.write()
			go client.read()
		} else {
			c.Header("Content-Type", "text/html; charset=utf-8")
			tailfTmpl.Execute(c.Writer, nil)
		}
	}
}

func (tf *tailF) read() {
	ticker := time.NewTicker(1 * time.Second)
	defer func() {
		ticker.Stop()
		tf.conn.Close()
	}()
	for {
		select {
		case <-ticker.C:
			tf.getData()
		}
	}
}

func (tf *tailF) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		tf.conn.Close()
	}()
	for {
		select {
		case <-ticker.C:
			tf.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := tf.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case message := <-tf.send:
			tf.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := tf.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

func (tf *tailF) getData() {
	if tf.filename == "" {
		tf.filename = "log.log"
	}

	file, err := os.Open(tf.filename)
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer file.Close()

	info, _ := file.Stat()
	size := info.Size()

	if tf.fileOffset == size {
		return
	}

	if tf.fileOffset == 0 && size > maxMessageSize*2 {
		tf.fileOffset = size - maxMessageSize*2
	}

	file.Seek(tf.fileOffset, 1)
	tf.fileOffset = size

	data := []byte{}
	for {
		buf := make([]byte, 1)
		_, err := file.Read(buf)
		if err != nil {
			break
		}
		data = append(data, buf...)
	}

	data = bytes.Replace(data, []byte("\x1b[0m"), []byte(""), -1)
	data = bytes.Replace(data, []byte("\x1b[31m"), []byte(""), -1)
	data = bytes.Replace(data, []byte("\x1b[32m"), []byte(""), -1)
	data = bytes.Replace(data, []byte("\x1b[33m"), []byte(""), -1)
	data = bytes.Replace(data, []byte("\x1b[34m"), []byte(""), -1)
	data = bytes.Replace(data, []byte("\x1b[35m"), []byte(""), -1)
	data = bytes.Replace(data, []byte("\x1b[36m"), []byte(""), -1)

	tf.send <- []byte(url.PathEscape(string(data)))
}
