package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strings"
	"syscall"
)

func isSOPSInvocation(args []string) bool {
	if strings.HasPrefix(args[0], "-") {
		return true
	}
	switch args[0] {
	case "encrypt", "decrypt", "edit", "rotate", "updatekeys", "unset", "exec-env", "exec-file", "filestatus", "groups", "keyservice", "publish", "completion", "help", "h":
		return true
	case "set":
		return len(args) > 1 && !slices.Contains(args, "-f") && !slices.Contains(args, "--env-file")
	default:
		info, err := os.Stat(args[0])
		return err == nil && info.Mode().IsRegular()
	}
}

func cmdSOPS(args []string, stdin io.Reader, stdout, stderr io.Writer, getenv func(string) string) int {
	bin, err := exec.LookPath("sops")
	if err != nil {
		fmt.Fprintln(stderr, "sops: install the SOPS CLI and put it on PATH to use standard SOPS commands")
		return 1
	}
	if getenv == nil {
		getenv = os.Getenv
	}
	getenv = withKeychainAgeKey(getenv)
	env := os.Environ()
	for _, key := range []string{"SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD"} {
		env = slices.DeleteFunc(env, func(entry string) bool { return strings.HasPrefix(entry, key+"=") })
		if value := getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	runner := childRunner{env: env, stdin: stdin, stdout: stdout, stderr: stderr, signals: signals}
	return runner.run(append([]string{bin}, args...))
}
