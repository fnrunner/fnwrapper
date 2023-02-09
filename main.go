package main

import (
	"context"
	"fmt"
	"os"

	fnrunv1alpha1 "github.com/fnrunner/fnruntime/apis/fnrun/v1alpha1"
	"github.com/fnrunner/fnwrapper/internal/exechandler"
	"github.com/fnrunner/fnwrapper/internal/grpcserver"
	"github.com/fnrunner/fnwrapper/internal/healthhandler"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
)

func main() {
	ws := &wrapperServer{
		l: ctrl.Log.WithName("fnwrapper"),
	}
	cmd := &cobra.Command{
		Use:   "fnwrapper",
		Short: "fnwrapper is a grpc server that frontends a fn that operates on KRM",
		RunE: func(cmd *cobra.Command, args []string) error {
			argsLenAtDash := cmd.ArgsLenAtDash()
			if argsLenAtDash > -1 {
				ws.entrypoint = args[argsLenAtDash:]
			}

			ctrl.SetLogger(zap.New())
			ws.l = ctrl.Log.WithName("fnwrapper")
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
	r.l.Info("start fnwrapper")
	wh := healthhandler.New()
	eh := exechandler.New(r.entrypoint)

	s := grpcserver.New(grpcserver.Config{
		Address:  fmt.Sprintf(":%d", r.port),
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
