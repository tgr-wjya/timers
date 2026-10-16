package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type Client struct {
	socketPath string
	conn       net.Conn
	reader     *bufio.Reader
}

func Dial(socketPath string) (*Client, error) {
	conn, err := net.DialTimeout("unix", socketPath, 150*time.Millisecond)
	if err != nil {
		return nil, err
	}

	return &Client{
		socketPath: socketPath,
		conn:       conn,
		reader:     bufio.NewReader(conn),
	}, nil
}

func (c *Client) Send(req Request) (Response, error) {
	var resp Response
	if c.conn == nil {
		return resp, fmt.Errorf("client not connected")
	}

	_ = c.conn.SetDeadline(time.Now().Add(5 * time.Second))
	data, err := json.Marshal(req)
	if err != nil {
		return resp, err
	}

	data = append(data, '\n')
	if _, err := c.conn.Write(data); err != nil {
		return resp, err
	}

	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return resp, err
	}

	if err := json.Unmarshal(line, &resp); err != nil {
		return resp, err
	}

	return resp, nil
}

func (c *Client) Ping() bool {
	resp, err := c.Send(Request{Action: "ping"})
	if err != nil {
		return false
	}
	return resp.Status == "ok"
}

func PingSocket(socketPath string) bool {
	client, err := Dial(socketPath)
	if err != nil {
		return false
	}
	defer client.Close()
	return client.Ping()
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
