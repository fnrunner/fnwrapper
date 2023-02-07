package main

import (
	"context"
	"fmt"
	fnrunv1alpha1 "github.com/fnrunner/fnruntime/apis/fnrun/v1alpha1"
	"github.com/fnrunner/fnwrapper/internal/exechandler"
	"github.com/fnrunner/fnwrapper/internal/grpcserver"
	"github.com/fnrunner/fnwrapper/internal/healthhandler"
	"os"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
)

func main() {
	ws := &wrapperServer{}
	cmd := &cobra.Command{
		Use:   "fnwrapper",
		Short: "fnwrapper is a grpc server that frontends a fn that operates on KRM",
		RunE: func(cmd *cobra.Command, args []string) error {
			argsLenAtDash := cmd.ArgsLenAtDash()
			if argsLenAtDash > -1 {
				ws.entrypoint = args[argsLenAtDash:]
			}
			return ws.run()
		},
	}
	cmd.Flags().IntVar(&ws.port, "port", fnrunv1alpha1.FnGRPCServerPort, "The server port")
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
		os.Exit(1)
	}
}

type wrapperServer struct {
	port       int
	entrypoint []string
	l          logr.Logger
}

func (r *wrapperServer) run() error {
	wh := healthhandler.New()
	eh := exechandler.New(r.entrypoint)
	s := grpcserver.New(grpcserver.Config{
		Insecure: true,
	},
		grpcserver.WithWatchHandler(wh.Watch),
		grpcserver.WithCheckHandler(wh.Check),
		grpcserver.WithExecHandler(eh.ExecuteFuntion),
	)

	if err := s.Start(context.Background()); err != nil {
		r.l.Error(err, "cannot start grpcserver")
		return err
	}

	return nil
}
