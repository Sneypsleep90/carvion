package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const MaxRPCMessageBytes = 1 << 20

const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeAdapterFailure = -32000
)

type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Serve processes exactly one JSON-RPC request framed as a single JSON line.
// The process manager starts one adapter process per operation and closes stdin
// after writing the request.
func Serve(ctx context.Context, input io.Reader, output io.Writer, adapter Adapter) error {
	message, err := io.ReadAll(io.LimitReader(input, MaxRPCMessageBytes+1))
	if err != nil {
		return fmt.Errorf("read RPC request: %w", err)
	}
	if len(message) == 0 || len(message) > MaxRPCMessageBytes {
		return errors.New("RPC request is empty or exceeds the size limit")
	}
	if message[len(message)-1] != '\n' {
		return errors.New("RPC request must end with a newline")
	}
	message = bytes.TrimSpace(message)

	var request RPCRequest
	if err := json.Unmarshal(message, &request); err != nil {
		return writeRPCResponse(output, RPCResponse{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: CodeParseError, Message: "invalid JSON request"},
		})
	}
	if request.JSONRPC != "2.0" || request.ID == "" || request.Method == "" {
		return writeRPCResponse(output, RPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error:   &RPCError{Code: CodeInvalidRequest, Message: "invalid JSON-RPC request"},
		})
	}

	result, rpcErr := dispatch(ctx, adapter, request)
	response := RPCResponse{JSONRPC: "2.0", ID: request.ID}
	if rpcErr != nil {
		response.Error = rpcErr
	} else {
		response.Result, err = json.Marshal(result)
		if err != nil {
			return fmt.Errorf("encode RPC result: %w", err)
		}
	}
	return writeRPCResponse(output, response)
}

func dispatch(ctx context.Context, adapter Adapter, request RPCRequest) (any, *RPCError) {
	if adapter == nil {
		return nil, &RPCError{Code: CodeAdapterFailure, Message: "adapter is unavailable"}
	}

	switch request.Method {
	case "GetInfo":
		var params struct{}
		if err := decodeParams(request.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		result, err := adapter.GetInfo(ctx)
		if err == nil {
			err = ValidateManifest(result)
		}
		return result, adapterError(err)
	case "Detect":
		var params DetectRequest
		if err := decodeParams(request.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		if len(params.Observation) == 0 {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "observation is required"}
		}
		result, err := adapter.Detect(ctx, params)
		if err != nil {
			return nil, adapterError(err)
		}
		if !validStatus(result.Status) {
			return nil, &RPCError{Code: CodeAdapterFailure, Message: "adapter returned an invalid detection status"}
		}
		return result, nil
	case "CheckCompatibility":
		var params CompatibilityRequest
		if err := decodeParams(request.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		result, err := adapter.CheckCompatibility(ctx, params)
		if err != nil {
			return nil, adapterError(err)
		}
		if !validStatus(result.Status) {
			return nil, &RPCError{Code: CodeAdapterFailure, Message: "adapter returned an invalid compatibility status"}
		}
		return result, nil
	case "InspectResources":
		var params InspectResourcesRequest
		if err := decodeParams(request.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		result, err := adapter.InspectResources(ctx, params)
		return result, adapterError(err)
	default:
		return nil, &RPCError{Code: CodeMethodNotFound, Message: "method not found"}
	}
}

func decodeParams(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func invalidParams(err error) *RPCError {
	return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
}

func adapterError(err error) *RPCError {
	if err == nil {
		return nil
	}
	return &RPCError{Code: CodeAdapterFailure, Message: err.Error()}
}

func validStatus(status CompatibilityStatus) bool {
	return status == StatusSupported || status == StatusUnsupported || status == StatusUnknown
}

func writeRPCResponse(output io.Writer, response RPCResponse) error {
	message, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("encode RPC response: %w", err)
	}
	if len(message) > MaxRPCMessageBytes {
		return errors.New("RPC response exceeds the size limit")
	}
	message = append(message, '\n')
	if _, err := output.Write(message); err != nil {
		return fmt.Errorf("write RPC response: %w", err)
	}
	return nil
}
