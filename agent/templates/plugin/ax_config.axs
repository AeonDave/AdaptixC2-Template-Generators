/// ax_config.axs — __NAME_CAP__ Agent UI and command definitions
/// AdaptixC2 custom agent registration script.

let exit_action = menu.create_action("Exit", function(agents_id) { agents_id.forEach(id => ax.execute_command(id, "exit")) });
let terminate_action = menu.create_action("Terminate", function(agents_id) { agents_id.forEach(id => ax.execute_command(id, "terminate")) });
menu.add_session_agent(exit_action, ["__NAME__"])
menu.add_session_agent(terminate_action, ["__NAME__"])

let file_browser_action     = menu.create_action("File Browser",    function(agents_id) { agents_id.forEach(id => ax.open_browser_files(id)) });
let process_browser_action  = menu.create_action("Process Browser", function(agents_id) { agents_id.forEach(id => ax.open_browser_process(id)) });
let terminal_browser_action = menu.create_action("Remote Terminal", function(agents_id) { agents_id.forEach(id => ax.open_remote_terminal(id)) });
menu.add_session_browser(file_browser_action, ["__NAME__"])
menu.add_session_browser(process_browser_action, ["__NAME__"])
menu.add_session_browser(terminal_browser_action, ["__NAME__"])

let tunnel_access_action = menu.create_action("Create Tunnel", function(agents_id) { ax.open_access_tunnel(agents_id[0], true, true, true, true) });
menu.add_session_access(tunnel_access_action, ["__NAME__"]);


let execute_action = menu.create_action("Execute", function(files_list) {
    file = files_list[0];
    if(file.type != "file"){ return; }

    let label_bin = form.create_label("Binary:");
    let text_bin = form.create_textline(file.path + file.name);
    text_bin.setEnabled(false);
    let label_args = form.create_label("Arguments:");
    let text_args = form.create_textline();

    let layout = form.create_gridlayout();
    layout.addWidget(label_bin, 0, 0, 1, 1);
    layout.addWidget(text_bin, 0, 1, 1, 1);
    layout.addWidget(label_args, 1, 0, 1, 1);
    layout.addWidget(text_args, 1, 1, 1, 1);

    let dialog = form.create_dialog("Execute binary");
    dialog.setSize(500, 80);
    dialog.setLayout(layout);
    if ( dialog.exec() == true )
    {
        let command = "run " + text_bin.text() + " " + text_args.text();
        ax.execute_command(file.agent_id, command);
    }
});
let download_action = menu.create_action("Download", function(files_list) { files_list.forEach( file => ax.execute_command(file.agent_id, "download " + file.path + file.name) ) });
let remove_action = menu.create_action("Remove", function(files_list) { files_list.forEach( file => ax.execute_command(file.agent_id, "rm " + file.path + file.name) ) });
menu.add_filebrowser(download_action, ["__NAME__"])
menu.add_filebrowser(remove_action, ["__NAME__"])


let job_stop_action = menu.create_action("Stop job", function(tasks_list) {
    tasks_list.forEach((task) => {
        if(task.type == "JOB" && task.state == "Running") {
            ax.execute_command(task.agent_id, "jobs kill " + task.task_id);
        }
    });
});
menu.add_tasks_job(job_stop_action, ["__NAME__"])



let cancel_action = menu.create_action("Cancel", function(files_list) { files_list.forEach( file => ax.execute_command(file.agent_id, "exfil cancel " + file.file_id) ) });
let resume_action = menu.create_action("Resume", function(files_list) { files_list.forEach( file => ax.execute_command(file.agent_id, "exfil start " + file.file_id) ) });
let pause_action  = menu.create_action("Pause",  function(files_list) { files_list.forEach( file => ax.execute_command(file.agent_id, "exfil stop " + file.file_id) ) });
menu.add_downloads_running(cancel_action, ["__NAME__"])
menu.add_downloads_running(resume_action, ["__NAME__"])
menu.add_downloads_running(pause_action, ["__NAME__"])



var event_files_action = function(id, path) {
    ax.execute_browser(id, "ls " + path);
}
event.on_filebrowser_list(event_files_action, ["__NAME__"]);

var event_upload_action = function(id, path, filepath) {
    let filename = ax.file_basename(filepath);
    ax.execute_browser(id, "upload " + filepath + " " + path + filename);
}
event.on_filebrowser_upload(event_upload_action, ["__NAME__"]);

var event_process_action = function(id) {
    ax.execute_browser(id, "ps");
}
event.on_processbrowser_list(event_process_action, ["__NAME__"]);

var event_disks_action = function(id) {
    ax.execute_browser(id, "disks");
}
event.on_filebrowser_disks(event_disks_action, ["__NAME__"]);



function RegisterCommands(listenerType)
{
    // ─── burst ──────────────────────────────────────────────────────────
    let _cmd_burst_show = ax.create_command("show", "Show current burst mode status", "burst show", "Task: show burst status");
    let _cmd_burst_set = ax.create_command("set", "Configure burst mode", "burst set 1 500 20", "Task: set burst mode");
    _cmd_burst_set.addArgInt("enabled", true, "1=enabled, 0=disabled");
    _cmd_burst_set.addArgInt("sleep", true, "Burst sleep in ms");
    _cmd_burst_set.addArgInt("jitter", true, "Burst jitter (0-100)");
    let cmd_burst = ax.create_command("burst", "Manage burst mode");
    cmd_burst.addSubCommands([_cmd_burst_show, _cmd_burst_set]);

    // ─── cat ────────────────────────────────────────────────────────────
    let cmd_cat_win = ax.create_command("cat", "Read a file (less 10 KB)", "cat C:\\file.exe", "Task: read file");
    cmd_cat_win.addArgString("path", true);
    let cmd_cat_unix = ax.create_command("cat", "Read a file (less 10 KB)", "cat /etc/passwd", "Task: read file");
    cmd_cat_unix.addArgString("path", true);

    // ─── cp ─────────────────────────────────────────────────────────────
    let cmd_cp = ax.create_command("cp", "Copy file or directory", "cp src.txt dst.txt", "Task: copy file or directory");
    cmd_cp.addArgString("src", true);
    cmd_cp.addArgString("dst", true);

    // ─── cd ─────────────────────────────────────────────────────────────
    let cmd_cd_win = ax.create_command("cd", "Change current working directory", "cd C:\\Windows", "Task: change working directory");
    cmd_cd_win.addArgString("path", true);
    let cmd_cd_unix = ax.create_command("cd", "Change current working directory", "cd /home/user", "Task: change working directory");
    cmd_cd_unix.addArgString("path", true);

    // ─── disks ──────────────────────────────────────────────────────────
    let cmd_disks = ax.create_command("disks", "List disk drives (Windows)", "disks", "Task: list disks");

    // ─── download ───────────────────────────────────────────────────────
    let cmd_download_win = ax.create_command("download", "Download a file", "download C:\\Temp\\file.txt", "Task: download file");
    cmd_download_win.addArgString("path", true);
    let cmd_download_unix = ax.create_command("download", "Download a file", "download /tmp/file", "Task: download file");
    cmd_download_unix.addArgString("path", true);

    // ─── execute ────────────────────────────────────────────────────────
    let _cmd_execute_bof = ax.create_command("bof", "Execute Beacon Object File", "execute bof /home/user/whoami.o", "Task: execute BOF");
    _cmd_execute_bof.addArgBool("-a", "Async mode");
    _cmd_execute_bof.addArgFile("bof", true, "Path to object file");
    _cmd_execute_bof.addArgString("param_data", false);
    let cmd_execute = ax.create_command("execute", "Execute [bof] in the current process's memory");
    cmd_execute.addSubCommands([_cmd_execute_bof])

    // ─── exfil ──────────────────────────────────────────────────────────
    let _cmd_exfil_cancel = ax.create_command("cancel", "Cancel a download", "exfil cancel abc123", "Task: cancel download");
    _cmd_exfil_cancel.addArgString("file_id", true, "File ID to cancel");
    let _cmd_exfil_start = ax.create_command("start", "Resume a stopped download", "exfil start abc123", "Task: resume download");
    _cmd_exfil_start.addArgString("file_id", true, "File ID to resume");
    let _cmd_exfil_stop = ax.create_command("stop", "Stop/pause a download", "exfil stop abc123", "Task: stop download");
    _cmd_exfil_stop.addArgString("file_id", true, "File ID to stop");
    let cmd_exfil = ax.create_command("exfil", "Download exfiltration management");
    cmd_exfil.addSubCommands([_cmd_exfil_cancel, _cmd_exfil_start, _cmd_exfil_stop]);

    // ─── exit ───────────────────────────────────────────────────────────
    let cmd_exit = ax.create_command("exit", "Kill agent", "exit", "Task: kill agent");

    // ─── getuid ─────────────────────────────────────────────────────────
    let cmd_getuid = ax.create_command("getuid", "Get current user identity", "getuid", "Task: get user identity");

    // ─── jobs ───────────────────────────────────────────────────────────
    let _cmd_job_list = ax.create_command("list", "List of jobs", "jobs list", "Task: show jobs");
    let _cmd_job_kill = ax.create_command("kill", "Kill a specified job", "jobs kill 1a2b3c4d", "Task: kill job");
    _cmd_job_kill.addArgString("task_id", true);
    let cmd_jobs = ax.create_command("jobs", "Long-running tasks manager");
    cmd_jobs.addSubCommands([_cmd_job_list, _cmd_job_kill]);

    // ─── kill ───────────────────────────────────────────────────────────
    let cmd_kill = ax.create_command("kill", "Kill a process with a given PID", "kill 7865", "Task: kill process");
    cmd_kill.addArgInt("pid", true);

    // ─── link ───────────────────────────────────────────────────────────
    let _cmd_link_smb = ax.create_command("smb", "Link to agent over SMB pipe", "link smb 10.0.0.5 mypipe", "Task: link SMB");
    _cmd_link_smb.addArgString("target", true, "Target host");
    _cmd_link_smb.addArgString("pipename", true, "Pipe name");
    let _cmd_link_tcp = ax.create_command("tcp", "Link to agent over TCP", "link tcp 10.0.0.5 4444", "Task: link TCP");
    _cmd_link_tcp.addArgString("target", true, "Target host");
    _cmd_link_tcp.addArgInt("port", true, "Target port");
    let cmd_link = ax.create_command("link", "Link to a child agent");
    cmd_link.addSubCommands([_cmd_link_smb, _cmd_link_tcp]);

    // ─── lportfwd ───────────────────────────────────────────────────────
    let _cmd_lportfwd_start = ax.create_command("start", "Start local port forward", "lportfwd start 0.0.0.0 8080 10.0.0.5 80", "Task: start lportfwd");
    _cmd_lportfwd_start.addArgFlagString("-h", "address", "Listening interface", "0.0.0.0");
    _cmd_lportfwd_start.addArgInt("lport", true, "Local port");
    _cmd_lportfwd_start.addArgString("fwdhost", true, "Forward host");
    _cmd_lportfwd_start.addArgInt("fwdport", true, "Forward port");
    let _cmd_lportfwd_stop = ax.create_command("stop", "Stop local port forward", "lportfwd stop 8080", "Task: stop lportfwd");
    _cmd_lportfwd_stop.addArgInt("lport", true, "Local port");
    let cmd_lportfwd = ax.create_command("lportfwd", "Local port forwarding");
    cmd_lportfwd.addSubCommands([_cmd_lportfwd_start, _cmd_lportfwd_stop]);

    // ─── ls ─────────────────────────────────────────────────────────────
    let cmd_ls_win = ax.create_command("ls", "List contents of a directory or details of a file", "ls C:\\Windows", "Task: list files");
    cmd_ls_win.addArgString("path", "", ".");
    let cmd_ls_unix = ax.create_command("ls", "List contents of a directory or details of a file", "ls /home/", "Task: list files");
    cmd_ls_unix.addArgString("path", "", ".");

    // ─── mv ─────────────────────────────────────────────────────────────
    let cmd_mv = ax.create_command("mv", "Move file or directory", "mv src.txt dst.txt", "Task: move file or directory");
    cmd_mv.addArgString("src", true);
    cmd_mv.addArgString("dst", true);

    // ─── mkdir ──────────────────────────────────────────────────────────
    let cmd_mkdir_win = ax.create_command("mkdir", "Make a directory", "mkdir C:\\Temp", "Task: make directory");
    cmd_mkdir_win.addArgString("path", true);
    let cmd_mkdir_unix = ax.create_command("mkdir", "Make a directory", "mkdir /tmp/ex", "Task: make directory");
    cmd_mkdir_unix.addArgString("path", true);

    // ─── profile ────────────────────────────────────────────────────────
    let _cmd_profile_chunk = ax.create_command("chunksize", "Set download chunk size", "profile chunksize 524288", "Task: set chunk size");
    _cmd_profile_chunk.addArgInt("size", true, "Chunk size in bytes");
    let _cmd_profile_killdate = ax.create_command("killdate", "Set agent kill date", "profile killdate 2025-12-31", "Task: set kill date");
    _cmd_profile_killdate.addArgString("date", true, "Date in YYYY-MM-DD format");
    let _cmd_profile_worktime = ax.create_command("workingtime", "Set agent working hours", "profile workingtime 08:00-17:00", "Task: set working time");
    _cmd_profile_worktime.addArgString("value", true, "Time range (e.g. 08:00-17:00)");
    let cmd_profile = ax.create_command("profile", "Agent profile settings");
    cmd_profile.addSubCommands([_cmd_profile_chunk, _cmd_profile_killdate, _cmd_profile_worktime]);

    // ─── ps ─────────────────────────────────────────────────────────────
    let cmd_ps = ax.create_command("ps", "Show process list", "ps", "Task: show process list");

    // ─── pwd ────────────────────────────────────────────────────────────
    let cmd_pwd = ax.create_command("pwd", "Print current working directory", "pwd", "Task: print working directory");

    // ─── rev2self ───────────────────────────────────────────────────────
    let cmd_rev2self = ax.create_command("rev2self", "Revert to your original access token", "rev2self", "Task: revert token");

    // ─── rm ─────────────────────────────────────────────────────────────
    let cmd_rm_win = ax.create_command("rm", "Remove a file or folder", "rm C:\\Temp\\file.txt", "Task: remove file or directory");
    cmd_rm_win.addArgString("path", true);
    let cmd_rm_unix = ax.create_command("rm", "Remove a file or folder", "rm /tmp/file", "Task: remove file or directory");
    cmd_rm_unix.addArgString("path", true);

    // ─── rportfwd ───────────────────────────────────────────────────────
    let _cmd_rportfwd_start = ax.create_command("start", "Start reverse port forward", "rportfwd start 8080 10.0.0.5 80", "Task: start rportfwd");
    _cmd_rportfwd_start.addArgInt("lport", true, "Remote listen port");
    _cmd_rportfwd_start.addArgString("fwdhost", true, "Forward host");
    _cmd_rportfwd_start.addArgInt("fwdport", true, "Forward port");
    let _cmd_rportfwd_stop = ax.create_command("stop", "Stop reverse port forward", "rportfwd stop 8080", "Task: stop rportfwd");
    _cmd_rportfwd_stop.addArgInt("lport", true, "Remote port");
    let cmd_rportfwd = ax.create_command("rportfwd", "Reverse port forwarding");
    cmd_rportfwd.addSubCommands([_cmd_rportfwd_start, _cmd_rportfwd_stop]);

    // ─── run ────────────────────────────────────────────────────────────
    let cmd_run_win = ax.create_command("run", "Execute long command or scripts", "run C:\\Windows\\cmd.exe /c \"whoami /all\"", "Task: command run");
    cmd_run_win.addArgString("program", true);
    cmd_run_win.addArgString("args", false);
    cmd_run_win.addArgBool("-s", "Steal token from target process (run as impersonated user)");
    let cmd_run_unix = ax.create_command("run", "Execute long command or scripts", "run /tmp/script.sh", "Task: command run");
    cmd_run_unix.addArgString("program", true);
    cmd_run_unix.addArgString("args", false);

    // ─── screenshot ─────────────────────────────────────────────────────
    let cmd_screenshot = ax.create_command("screenshot", "Take a single screenshot", "screenshot", "Task: screenshot");

    // ─── sleep ──────────────────────────────────────────────────────────
    let cmd_sleep = ax.create_command("sleep", "Set agent sleep interval and jitter", "sleep 30 10", "Task: set sleep");
    cmd_sleep.addArgString("value", true, "Sleep interval (seconds or duration like 5m30s)");
    cmd_sleep.addArgInt("jitter", false, "Jitter percentage (0-100)");

    // ─── socks ──────────────────────────────────────────────────────────
    let _cmd_socks_start = ax.create_command("start", "Start a SOCKS5 proxy server and listen on a specified port", "socks start 1080 -a user pass");
    _cmd_socks_start.addArgFlagString("-h", "address", "Listening interface address", "0.0.0.0");
    _cmd_socks_start.addArgInt("port", true, "Listen port");
    _cmd_socks_start.addArgBool("-a", "Enable User/Password authentication for SOCKS5");
    _cmd_socks_start.addArgString("username", false, "Username for SOCKS5 proxy");
    _cmd_socks_start.addArgString("password", false, "Password for SOCKS5 proxy");
    let _cmd_socks_stop = ax.create_command("stop", "Stop a SOCKS proxy server", "socks stop 1080");
    _cmd_socks_stop.addArgInt("port", true);
    let cmd_socks = ax.create_command("socks", "Managing socks tunnels");
    cmd_socks.addSubCommands([_cmd_socks_start, _cmd_socks_stop]);

    // ─── interact ───────────────────────────────────────────────────────
    let cmd_interact = ax.create_command("interact", "Enter interactive mode (sleep 0)", "interact", "Task: interact");

    // ─── powershell ─────────────────────────────────────────────────────
    let cmd_powershell = ax.create_command("powershell", "Execute command via powershell.exe", "powershell Get-Process", "Task: command execute");
    cmd_powershell.addArgString("cmd", true);

    // ─── shell ──────────────────────────────────────────────────────────
    let cmd_shell_win = ax.create_command("shell", "Execute command via cmd.exe", "shell whoami /all", "Task: command execute");
    cmd_shell_win.addArgString("cmd", true);
    let cmd_shell_unix = ax.create_command("shell", "Execute command via /bin/sh", "shell id", "Task: command execute");
    cmd_shell_unix.addArgString("cmd", true);

    // ─── terminate ──────────────────────────────────────────────────────
    let cmd_terminate = ax.create_command("terminate", "Terminate the agent", "terminate", "Task: terminate agent");

    // ─── unlink ─────────────────────────────────────────────────────────
    let cmd_unlink = ax.create_command("unlink", "Unlink from a child pivot agent", "unlink mypivot", "Task: unlink pivot");
    cmd_unlink.addArgString("name", true, "Pivot name");

    // ─── upload ─────────────────────────────────────────────────────────
    let cmd_upload_win = ax.create_command("upload", "Upload a file", "upload /tmp/file.txt C:\\Temp\\file.txt", "Task: upload file");
    cmd_upload_win.addArgFile("local_file", true);
    cmd_upload_win.addArgString("remote_path", false);
    let cmd_upload_unix = ax.create_command("upload", "Upload a file", "upload /tmp/file.txt /root/file.txt", "Task: upload file");
    cmd_upload_unix.addArgFile("local_file", true);
    cmd_upload_unix.addArgString("remote_path", false);

    // ─── zip ────────────────────────────────────────────────────────────
    let cmd_zip_win = ax.create_command("zip", "Archive (zip) a file or directory", "zip C:\\backup C:\\Temp\\qwe.zip", "Task: Zip a file or directory");
    cmd_zip_win.addArgString("path", true);
    cmd_zip_win.addArgString("zip_path", true);
    let cmd_zip_unix = ax.create_command("zip", "Archive (zip) a file or directory", "zip /home/test /tmp/qwe.zip", "Task: Zip a file or directory");
    cmd_zip_unix.addArgString("path", true);
    cmd_zip_unix.addArgString("zip_path", true);

    // ─── Command groups ─────────────────────────────────────────────────
    let commands_win  = ax.create_commands_group("__NAME__", [cmd_burst, cmd_cat_win,  cmd_cp, cmd_cd_win,  cmd_disks, cmd_download_win,  cmd_execute, cmd_exfil, cmd_exit, cmd_getuid, cmd_interact, cmd_jobs, cmd_kill, cmd_link, cmd_lportfwd, cmd_ls_win,  cmd_mv, cmd_mkdir_win,  cmd_powershell, cmd_profile, cmd_ps, cmd_pwd, cmd_rev2self, cmd_rm_win,  cmd_rportfwd, cmd_run_win,  cmd_screenshot, cmd_sleep, cmd_socks, cmd_shell_win,  cmd_terminate, cmd_unlink, cmd_upload_win,  cmd_zip_win] );
    let commands_unix = ax.create_commands_group("__NAME__", [cmd_burst, cmd_cat_unix, cmd_cp, cmd_cd_unix,            cmd_download_unix,             cmd_exfil, cmd_exit, cmd_getuid, cmd_interact, cmd_jobs, cmd_kill, cmd_link, cmd_lportfwd, cmd_ls_unix, cmd_mv, cmd_mkdir_unix, cmd_profile, cmd_ps, cmd_pwd,               cmd_rm_unix, cmd_rportfwd, cmd_run_unix, cmd_screenshot, cmd_sleep, cmd_socks, cmd_shell_unix, cmd_terminate, cmd_unlink, cmd_upload_unix, cmd_zip_unix] );

    return {
        commands_windows: commands_win,
        commands_linux: commands_unix,
        commands_macos: commands_unix
    }
}

function GenerateUI(listeners_type)
{
    let labelOS = form.create_label("OS:");
    let comboOS = form.create_combo()
    comboOS.addItems(["windows", "linux", "macos"]);

    let labelArch = form.create_label("Arch:");
    let comboArch = form.create_combo()
    comboArch.addItems(["amd64", "arm64"]);

    let labelFormat = form.create_label("Format:");
    let comboFormat = form.create_combo()
    comboFormat.addItems(["Exe", "Service Exe", "DLL", "Shellcode"]);

    let labelSvcName = form.create_label("Service name:");
    let textSvcName = form.create_textline("");
    textSvcName.setPlaceholder("MyService");
    labelSvcName.setVisible(false);
    textSvcName.setVisible(false);

    let checkWin7 = form.create_check("Windows 7 support");

    let hline = form.create_hline()

    let labelReconnTimeout = form.create_label("Reconnect timeout:");
    let textReconnTimeout = form.create_textline("10");
    textReconnTimeout.setPlaceholder("seconds")

    let labelReconnCount = form.create_label("Reconnect count:");
    let spinReconnCount = form.create_spin();
    spinReconnCount.setRange(0, 1000000000);
    spinReconnCount.setValue(1000000000);

    let hline2 = form.create_hline()

    let labelProxyHost = form.create_label("Proxy host:");
    let textProxyHost = form.create_textline("");
    textProxyHost.setPlaceholder("hostname or IP (empty = no proxy)")

    let labelProxyPort = form.create_label("Proxy port:");
    let spinProxyPort = form.create_spin();
    spinProxyPort.setRange(0, 65535);
    spinProxyPort.setValue(0);

    let labelProxyUser = form.create_label("Proxy user:");
    let textProxyUser = form.create_textline("");
    textProxyUser.setPlaceholder("optional")

    let labelProxyPass = form.create_label("Proxy password:");
    let textProxyPass = form.create_textline("");
    textProxyPass.setPlaceholder("optional")

    let layout = form.create_gridlayout();
    layout.addWidget(labelOS, 0, 0, 1, 1);
    layout.addWidget(comboOS, 0, 1, 1, 1);
    layout.addWidget(labelArch, 1, 0, 1, 1);
    layout.addWidget(comboArch, 1, 1, 1, 1);
    layout.addWidget(labelFormat, 2, 0, 1, 1);
    layout.addWidget(comboFormat, 2, 1, 1, 1);
    layout.addWidget(labelSvcName, 3, 0, 1, 1);
    layout.addWidget(textSvcName, 3, 1, 1, 1);
    layout.addWidget(checkWin7, 4, 1, 1, 1);
    layout.addWidget(hline, 5, 0, 1, 2);
    layout.addWidget(labelReconnTimeout, 6, 0, 1, 1);
    layout.addWidget(textReconnTimeout, 6, 1, 1, 1);
    layout.addWidget(labelReconnCount, 7, 0, 1, 1);
    layout.addWidget(spinReconnCount, 7, 1, 1, 1);
    layout.addWidget(hline2, 8, 0, 1, 2);
    layout.addWidget(labelProxyHost, 9, 0, 1, 1);
    layout.addWidget(textProxyHost, 9, 1, 1, 1);
    layout.addWidget(labelProxyPort, 10, 0, 1, 1);
    layout.addWidget(spinProxyPort, 10, 1, 1, 1);
    layout.addWidget(labelProxyUser, 11, 0, 1, 1);
    layout.addWidget(textProxyUser, 11, 1, 1, 1);
    layout.addWidget(labelProxyPass, 12, 0, 1, 1);
    layout.addWidget(textProxyPass, 12, 1, 1, 1);

    // ── Kill Date (optional — remove this block if not needed) ─────────
    let hline3 = form.create_hline()

    let checkKillDate = form.create_check("Set kill date");
    let labelKillDate = form.create_label("Kill date:");
    let dateKillDate = form.create_dateline();
    let _tomorrow = new Date(); _tomorrow.setDate(_tomorrow.getDate() + 1);
    dateKillDate.setDateString(("0" + _tomorrow.getDate()).slice(-2) + "." + ("0" + (_tomorrow.getMonth()+1)).slice(-2) + "." + _tomorrow.getFullYear());
    dateKillDate.setVisible(false);
    let labelKillTime = form.create_label("Kill time:");
    let timeKillTime = form.create_timeline();
    timeKillTime.setVisible(false);
    labelKillDate.setVisible(false);
    labelKillTime.setVisible(false);

    form.connect(checkKillDate, "stateChanged", function() {
        let checked = checkKillDate.isChecked();
        labelKillDate.setVisible(checked);
        dateKillDate.setVisible(checked);
        labelKillTime.setVisible(checked);
        timeKillTime.setVisible(checked);
    });

    // ── Working Time (optional — remove this block if not needed) ──────
    let checkWorkTime = form.create_check("Set working time");
    let labelStartTime = form.create_label("Start time:");
    let timeStartTime = form.create_timeline();
    timeStartTime.setVisible(false);
    let labelEndTime = form.create_label("End time:");
    let timeEndTime = form.create_timeline();
    timeEndTime.setVisible(false);
    labelStartTime.setVisible(false);
    labelEndTime.setVisible(false);

    form.connect(checkWorkTime, "stateChanged", function() {
        let checked = checkWorkTime.isChecked();
        labelStartTime.setVisible(checked);
        timeStartTime.setVisible(checked);
        labelEndTime.setVisible(checked);
        timeEndTime.setVisible(checked);
    });

    layout.addWidget(hline3, 13, 0, 1, 2);
    layout.addWidget(checkKillDate, 14, 1, 1, 1);
    layout.addWidget(labelKillDate, 15, 0, 1, 1);
    layout.addWidget(dateKillDate, 15, 1, 1, 1);
    layout.addWidget(labelKillTime, 16, 0, 1, 1);
    layout.addWidget(timeKillTime, 16, 1, 1, 1);
    layout.addWidget(checkWorkTime, 17, 1, 1, 1);
    layout.addWidget(labelStartTime, 18, 0, 1, 1);
    layout.addWidget(timeStartTime, 18, 1, 1, 1);
    layout.addWidget(labelEndTime, 19, 0, 1, 1);
    layout.addWidget(timeEndTime, 19, 1, 1, 1);

    form.connect(comboOS, "currentTextChanged", function(text) {
        comboFormat.clear();
        if(text == "windows") {
            comboFormat.addItems(["Exe", "Service Exe", "DLL", "Shellcode"]);
            checkWin7.setVisible(true);
        }
        else if (text == "linux") {
            comboFormat.addItems(["Binary", "Shared Object (.so)"]);
            checkWin7.setVisible(false);
        }
        else {
            comboFormat.addItems(["Binary Mach-O", "Dynamic Library (.dylib)"]);
            checkWin7.setVisible(false);
        }
    });

    form.connect(comboFormat, "currentTextChanged", function(text) {
        let isSvc = (text == "Service Exe");
        labelSvcName.setVisible(isSvc);
        textSvcName.setVisible(isSvc);
    });

    let container = form.create_container()
    container.put("os", comboOS)
    container.put("arch", comboArch)
    container.put("format", comboFormat)
    container.put("svc_name", textSvcName)
    container.put("reconn_timeout", textReconnTimeout)
    container.put("reconn_count", spinReconnCount)
    container.put("win7_support", checkWin7)
    container.put("proxy_host", textProxyHost)
    container.put("proxy_port", spinProxyPort)
    container.put("proxy_user", textProxyUser)
    container.put("proxy_pass", textProxyPass)
    container.put("is_killdate", checkKillDate)
    container.put("kill_date", dateKillDate)
    container.put("kill_time", timeKillTime)
    container.put("is_workingtime", checkWorkTime)
    container.put("start_time", timeStartTime)
    container.put("end_time", timeEndTime)

    let innerPanel = form.create_panel()
    innerPanel.setLayout(layout)

    let scroll = form.create_scrollarea()
    scroll.setPanel(innerPanel)

    let outerLayout = form.create_vlayout()
    outerLayout.addWidget(scroll)

    let outerPanel = form.create_panel()
    outerPanel.setLayout(outerLayout)

    return {
        ui_panel: outerPanel,
        ui_container: container,
        ui_height: 600,
        ui_width: 800
    }
}
