package signals

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Result is a finished command. A nil *Result means it did not start or did not finish in time.
type Result struct {
	Code int
	Out  string
}

// which finds name on the PATH of env, not of this process.
func which(name string, env map[string]string) string {
	for _, dir := range filepath.SplitList(env["PATH"]) {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return p
		}
	}
	return ""
}

func envList(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

// RunAll starts every command at once and returns what finished within timeout in total.
func RunAll(cmds map[string][]string, cwd string, timeout time.Duration, env map[string]string) map[string]*Result {
	deadline := time.Now().Add(timeout)
	type running struct {
		cmd  *exec.Cmd
		sink *os.File
		done chan struct{}
	}
	procs := map[string]*running{}
	out := map[string]*Result{}
	for name, argv := range cmds {
		out[name] = nil
		path := argv[0]
		if !strings.Contains(path, "/") {
			if path = which(path, env); path == "" {
				continue
			}
		}
		// Output goes to a file, not a pipe: a child that keeps a pipe open would block the read.
		sink, err := os.CreateTemp("", "builder-signal-*")
		if err != nil {
			continue
		}
		os.Remove(sink.Name())
		cmd := exec.Command(path, argv[1:]...)
		cmd.Dir, cmd.Env, cmd.Stdout = cwd, envList(env), sink
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if cmd.Start() != nil {
			sink.Close()
			continue
		}
		r := &running{cmd, sink, make(chan struct{})}
		go func() { r.cmd.Wait(); close(r.done) }()
		procs[name] = r
	}
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	for name, r := range procs {
		select {
		case <-r.done:
		case <-timer.C:
			timer.Reset(0)
		}
		select {
		case <-r.done:
			r.sink.Seek(0, io.SeekStart)
			b, _ := io.ReadAll(r.sink)
			out[name] = &Result{r.cmd.ProcessState.ExitCode(), strings.ToValidUTF8(string(b), "�")}
		default:
			// Kill the process group (the tool and any child it left), then reap it without waiting long.
			syscall.Kill(-r.cmd.Process.Pid, syscall.SIGKILL)
			select {
			case <-r.done:
			case <-time.After(200 * time.Millisecond):
			}
		}
		r.sink.Close()
	}
	return out
}
