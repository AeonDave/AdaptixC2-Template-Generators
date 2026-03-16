package protocol

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/vmihailenco/msgpack/v5"
)

// ─── Codec ─────────────────────────────────────────────────────────────────────

func Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}

// ─── Command codes (shared with protocol overlays) ─────────────────────────────
// Agent-side command constants (COMMAND_FS_*, COMMAND_OS_*, RESP_*, etc.) are
// in agent_types.go which survives protocol overlay replacement.

const (
	COMMAND_EXIT = 0

	COMMAND_EXEC_BOF       = 50
	COMMAND_EXEC_BOF_OUT   = 51
	COMMAND_EXEC_BOF_ASYNC = 52

	COMMAND_JOB_LIST = 60
	COMMAND_JOB_KILL = 61
)

// ─── Wire types ────────────────────────────────────────────────────────────────

type Profile struct {
	Type        uint     `msgpack:"type"`
	Addresses   []string `msgpack:"addresses"`
	BannerSize  int      `msgpack:"banner_size"`
	ConnTimeout int      `msgpack:"conn_timeout"`
	ConnCount   int      `msgpack:"conn_count"`
	UseSSL      bool     `msgpack:"use_ssl"`
	SslCert     []byte   `msgpack:"ssl_cert"`
	SslKey      []byte   `msgpack:"ssl_key"`
	CaCert      []byte   `msgpack:"ca_cert"`
	Sleep       int      `msgpack:"sleep"`
	Jitter      int      `msgpack:"jitter"`
	KillDate    int64    `msgpack:"kill_date"`
	WorkStart   int      `msgpack:"work_start"`
	WorkEnd     int      `msgpack:"work_end"`
}

type SessionInfo struct {
	Process    string `msgpack:"process"`
	PID        int    `msgpack:"pid"`
	User       string `msgpack:"user"`
	Host       string `msgpack:"host"`
	Ipaddr     string `msgpack:"ipaddr"`
	Elevated   bool   `msgpack:"elevated"`
	Acp        uint32 `msgpack:"acp"`
	Oem        uint32 `msgpack:"oem"`
	Os         string `msgpack:"os"`
	OSVersion  string `msgpack:"os_version"`
	EncryptKey []byte `msgpack:"encrypt_key"`
}

type Message struct {
	Type   int8     `msgpack:"type"`
	Object [][]byte `msgpack:"object"`
}

type Command struct {
	Code uint   `msgpack:"code"`
	Id   uint   `msgpack:"id"`
	Data []byte `msgpack:"data"`
}

// ─── Error type (shared with protocol overlays) ───────────────────────────────

type AnsError struct {
	Error string `msgpack:"error"`
}

// ─── Exfil type ────────────────────────────────────────────────────────────────

type AnsExfil struct {
	CommandId uint   `msgpack:"command_id"`
	Data      []byte `msgpack:"data"`
}

// ─── BOF types (shared with protocol overlays) ─────────────────────────────────

type ParamsExecBof struct {
	Object   []byte `msgpack:"object"`
	ArgsPack string `msgpack:"argspack"`
	Task     string `msgpack:"task"`
}

type BofMsg struct {
	Type int    `msgpack:"type"`
	Data []byte `msgpack:"data"`
}

type AnsExecBof struct {
	Msgs []byte `msgpack:"msgs"`
}

type AnsExecBofAsync struct {
	Msgs   []byte `msgpack:"msgs"`
	Start  bool   `msgpack:"start"`
	Finish bool   `msgpack:"finish"`
}

type JobInfo struct {
	JobId   string `msgpack:"job_id"`
	JobType int    `msgpack:"job_type"`
}

type AnsJobList struct {
	List []byte `msgpack:"list"`
}

const (
	BOF_PACK = 102

	CALLBACK_OUTPUT      = 0x0
	CALLBACK_OUTPUT_OEM  = 0x1e
	CALLBACK_OUTPUT_UTF8 = 0x20
	CALLBACK_ERROR       = 0x0d

	CALLBACK_AX_SCREENSHOT   = 0x81
	CALLBACK_AX_DOWNLOAD_MEM = 0x82
)

// ─── Network framing ───────────────────────────────────────────────────────────

// ConnRead reads exactly 'size' bytes from conn.
func ConnRead(conn net.Conn, size int) ([]byte, error) {
	buf := make([]byte, size)
	total := 0
	for total < size {
		n, err := conn.Read(buf[total:])
		if err != nil {
			return nil, err
		}
		total += n
	}
	return buf, nil
}

// RecvMsg reads a length-prefixed message (4-byte BE uint32 + payload).
func RecvMsg(conn net.Conn) ([]byte, error) {
	hdr, err := ConnRead(conn, 4)
	if err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(hdr)
	if size == 0 {
		return nil, nil
	}
	return ConnRead(conn, int(size))
}

// SendMsg writes a length-prefixed message (4-byte BE uint32 + payload).
func SendMsg(conn net.Conn, data []byte) error {
	hdr := make([]byte, 4)
	binary.BigEndian.PutUint32(hdr, uint32(len(data)))
	_, err := conn.Write(append(hdr, data...))
	return err
}

// ─── Zip helpers ───────────────────────────────────────────────────────────────

func ZipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("data")
	if err != nil {
		return nil, err
	}
	if _, err = f.Write(data); err != nil {
		return nil, err
	}
	if err = w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnzipBytes(data []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	if len(r.File) == 0 {
		return nil, fmt.Errorf("empty zip archive")
	}
	rc, err := r.File[0].Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}
