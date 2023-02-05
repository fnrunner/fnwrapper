package exechandler

import (
	"bytes"
	"context"
	"errors"
	"os/exec"

	"github.com/fnrunner/fnproto/pkg/executor/executorpb"
	"github.com/fnrunner/fnsdk/go/fn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *subServer) ExecuteFuntion(ctx context.Context, req *executorpb.ExecuteFunctionRequest) (*executorpb.ExecuteFunctionResponse, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, r.entrypoint[0], r.entrypoint[1:]...)
	cmd.Stdin = bytes.NewReader(req.ResourceContext)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	var exitErr *exec.ExitError
	outbytes := stdout.Bytes()
	stderrStr := stderr.String()
	if err != nil {
		if errors.As(err, &exitErr) {
			// If the exit code is non-zero, we will try to embed the structured results and content from stderr into the error message.
			rCtx, pe := fn.ParseResourceContext(outbytes)
			if pe != nil {
				// If we can't parse the output resource context, we only surface the content in stderr.
				return nil, status.Errorf(codes.Internal, "failed to execute function %q with stderr %v", req.Image, stderrStr)
			}
			return nil, status.Errorf(codes.Internal, "failed to execute function %q with structured results: %v and stderr: %v", req.Image, rCtx.Results.Error(), stderrStr)
		} else {
			return nil, status.Errorf(codes.Internal, "Failed to execute function %q: %s (%s)", req.Image, err, stderrStr)
		}
	}

	return &executorpb.ExecuteFunctionResponse{
		ResourceContext: outbytes,
		Log:             []byte(stderrStr),
	}, nil
}
