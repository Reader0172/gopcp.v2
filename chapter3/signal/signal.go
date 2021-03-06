package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func main() {
	//发送信号
	go func() {
		time.Sleep(5 * time.Second)
		sendSignal()
	}()
	//处理进程信号
	handleSignal()
}

func handleSignal() {
	//处理信号程序，会一直等待信号的到来
	sigRecv1 := make(chan os.Signal, 1)
	sigs1 := []os.Signal{syscall.SIGINT, syscall.SIGQUIT}
	fmt.Printf("Set notification for %s... [sigRecv1]\n", sigs1)
	signal.Notify(sigRecv1, sigs1...)
	sigRecv2 := make(chan os.Signal, 1)
	sigs2 := []os.Signal{syscall.SIGQUIT}
	fmt.Printf("Set notification for %s... [sigRecv2]\n", sigs2)
	signal.Notify(sigRecv2, sigs2...)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		//从通道等待信号
		for sig := range sigRecv1 {
			fmt.Printf("Received a signal from sigRecv1: %s\n", sig)
		}
		fmt.Printf("End. [sigRecv1]\n")
		wg.Done()
	}()
	go func() {
		//从通道等待信号
		for sig := range sigRecv2 {
			fmt.Printf("Received a signal from sigRecv2: %s\n", sig)
		}
		fmt.Printf("End. [sigRecv2]\n")
		wg.Done()
	}()

	fmt.Println("Wait for 25 seconds... ")
	time.Sleep(25 * time.Second)
	fmt.Printf("Stop notification...")
	signal.Stop(sigRecv1)
	close(sigRecv1)
	fmt.Printf("done. [sigRecv1]\n")
	//关闭sigRecv2
	fmt.Println("Wait for 26 seconds... ")
	time.Sleep(26 * time.Second)
	fmt.Printf("Stop notification...")
	signal.Stop(sigRecv2)
	close(sigRecv2)
	fmt.Printf("done. [sigRecv2]\n")
	wg.Wait()
}

func sendSignal() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("Fatal Error: %s\n", err)
			//打印错误调用栈
			debug.PrintStack()
		}
	}()
	// ps aux | grep "signal" | grep -v "grep" | grep -v "go run" | awk '{print $2}'
	//匿名管道
	cmds := []*exec.Cmd{
		exec.Command("ps", "aux"),
		exec.Command("grep", "signal"),
		exec.Command("grep", "-v", "grep"),
		exec.Command("grep", "-v", "go run"),
		exec.Command("awk", "{print $2}"),
	}
	output, err := runCmds(cmds)
	if err != nil {
		fmt.Printf("Command Execution Error: %s\n", err)
		return
	}
	//fmt.Println("output:",output)
	pids, err := getPids(output)
	fmt.Println("pids:",pids)
	if err != nil {
		fmt.Printf("PID Parsing Error: %s\n", err)
		return
	}
	fmt.Printf("Target PID(s):\n%v\n", pids)
	for _, pid := range pids {
		//根据进程ID获取进程值
		proc, err := os.FindProcess(pid)
		if err != nil {
			fmt.Printf("Process Finding Error: %s\n", err)
			return
		}
		sig := syscall.SIGQUIT
		fmt.Printf("Send signal '%s' to the process (pid=%d)...\n", sig, pid)
		//往进程发送信号
		err = proc.Signal(sig)
		if err != nil {
			fmt.Printf("Signal Sending Error: %s\n", err)
			return
		}
	}
}

func getPids(strs []string) ([]int, error) {
	var pids []int
	for _, str := range strs {
		fmt.Println("getPids str:",str)
		pid, err := strconv.Atoi(strings.TrimSpace(str))
		if err != nil {
			return nil, err
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

func runCmds(cmds []*exec.Cmd) ([]string, error) {
	if cmds == nil || len(cmds) == 0 {
		return nil, errors.New("The cmd slice is invalid!")
	}
	first := true
	var output []byte
	var err error
	for _, cmd := range cmds {
		fmt.Printf("Run command: %v\n", getCmdPlaintext(cmd))
		//把前一个命令的输出当做第二个命令的输入
		//第一个命令不会执行，因为第一个命令没有输入
		if !first {
			var stdinBuf bytes.Buffer
			stdinBuf.Write(output)
			cmd.Stdin = &stdinBuf
		}
		var stdoutBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		if err = cmd.Start(); err != nil {
			return nil, getError(err, cmd)
		}
		if err = cmd.Wait(); err != nil {
			return nil, getError(err, cmd)
		}
		//将输出buf里面的内容以字节形式返回
		output = stdoutBuf.Bytes()
		//fmt.Printf("Output:\n%s\n", string(output))
		//把上一个命令的输出当做下一个命令的输入，
		if first {
			first = false
		}
	}
	var lines []string
	var outputBuf bytes.Buffer
	outputBuf.Write(output)
	for {
		line, err := outputBuf.ReadBytes('\n')
		fmt.Println("line:",string(line))
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, getError(err, nil)
			}
		}
		lines = append(lines, string(line))
	}
	return lines, nil
}

func getCmdPlaintext(cmd *exec.Cmd) string {
	var buf bytes.Buffer
	buf.WriteString(cmd.Path)
	for _, arg := range cmd.Args[1:] {
		buf.WriteRune(' ')
		buf.WriteString(arg)
	}
	return buf.String()
}

func getError(err error, cmd *exec.Cmd, extraInfo ...string) error {
	var errMsg string
	if cmd != nil {
		errMsg = fmt.Sprintf("%s  [%s %v]", err, (*cmd).Path, (*cmd).Args)
	} else {
		errMsg = fmt.Sprintf("%s", err)
	}
	if len(extraInfo) > 0 {
		errMsg = fmt.Sprintf("%s (%v)", errMsg, extraInfo)
	}
	return errors.New(errMsg)
}
