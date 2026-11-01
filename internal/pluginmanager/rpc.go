package pluginmanager

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	sdk "github.com/Sneypsleep90/carvion/pkg/adaptersdk/v1"
)

const maxCapturedStderr = 16 << 10

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type limitedBuffer struct {
	mu    sync.Mutex
	limit int
	data  []byte
}

func runRPC(ctx context.Context, registered *plugin, method string, params, result any) (string, error) {
	request, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: "1", Method: method, Params: params})
	if err != nil {
		return "", fmt.Errorf("encode RPC request: %w", err)
	}
	if len(request) > sdk.MaxRPCMessageBytes {
		return "", errors.New("RPC request exceeds the size limit")
	}
	request = append(request, '\n')

	command := exec.CommandContext(ctx, registered.executable)
	command.Dir = registered.directory
	command.Env = sanitizedEnvironment()
	command.WaitDelay = 500 * time.Millisecond
	stdout, err := command.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("create adapter stdout pipe: %w", err)
	}
	stdin, err := command.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("create adapter stdin pipe: %w", err)
	}
	stderr := &limitedBuffer{limit: maxCapturedStderr}
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		return stderr.String(), fmt.Errorf("start adapter: %w", err)
	}

	if _, err := stdin.Write(request); err != nil {
		stopProcess(command)
		return stderr.String(), fmt.Errorf("write adapter request: %w", err)
	}
	if err := stdin.Close(); err != nil {
		stopProcess(command)
		return stderr.String(), fmt.Errorf("close adapter stdin: %w", err)
	}

	reader := bufio.NewReaderSize(stdout, sdk.MaxRPCMessageBytes+1)
	frame, err := reader.ReadSlice('\n')
	if err != nil {
		stopProcess(command)
		return stderr.String(), fmt.Errorf("read adapter response frame: %w", err)
	}
	if len(frame)-1 > sdk.MaxRPCMessageBytes {
		stopProcess(command)
		return stderr.String(), errors.New("adapter response exceeds the size limit")
	}
	trailing, err := io.ReadAll(io.LimitReader(reader, sdk.MaxRPCMessageBytes+1))
	if err != nil {
		stopProcess(command)
		return stderr.String(), fmt.Errorf("read adapter trailing output: %w", err)
	}
	if len(trailing) != 0 {
		stopProcess(command)
		return stderr.String(), errors.New("adapter wrote more than one response frame")
	}
	if err := command.Wait(); err != nil {
		return stderr.String(), fmt.Errorf("adapter process exited unsuccessfully: %w", err)
	}

	var response sdk.RPCResponse
	if err := json.Unmarshal(bytes.TrimSpace(frame), &response); err != nil {
		return stderr.String(), fmt.Errorf("decode adapter response: %w", err)
	}
	if response.JSONRPC != "2.0" || response.ID != "1" {
		return stderr.String(), errors.New("adapter response has an invalid JSON-RPC version or ID")
	}
	if response.Error != nil {
		return stderr.String(), fmt.Errorf("adapter RPC error %d: %s", response.Error.Code, response.Error.Message)
	}
	if len(response.Result) == 0 {
		return stderr.String(), errors.New("adapter response is missing a result")
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return stderr.String(), fmt.Errorf("decode adapter result: %w", err)
	}
	return stderr.String(), nil
}

func sanitizedEnvironment() []string {
	allowed := []string{"PATH", "LANG", "LC_ALL", "TMPDIR"}
	environment := make([]string, 0, len(allowed))
	for _, key := range allowed {
		if value, ok := os.LookupEnv(key); ok {
			environment = append(environment, key+"="+value)
		}
	}
	return environment
}

func stopProcess(command *exec.Cmd) {
	if command.Process != nil {
		_ = command.Process.Kill()
	}
	_ = command.Wait()
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.limit - len(b.data)
	if remaining > 0 {
		if remaining > len(data) {
			remaining = len(data)
		}
		b.data = append(b.data, data[:remaining]...)
	}
	return len(data), nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.TrimSpace(string(b.data))
}
