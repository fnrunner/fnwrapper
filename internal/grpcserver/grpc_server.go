/*
Copyright 2022 Nokia.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpcserver

import (
	"context"
	"net"

	"github.com/fnrunner/fnproto/pkg/executor/executorpb"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type GrpcServer struct {
	config Config
	executorpb.UnimplementedFunctionExecutorServer

	sem *semaphore.Weighted

	// logger
	l logr.Logger

	//Exec Handlers
	execHandler ExecHandler

	//health handlers
	checkHandler CheckHandler
	watchHandler WatchHandler
	//
	// cached certificate
	//cm *sync.Mutex
}

// Health Handlers
type CheckHandler func(context.Context, *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error)

type WatchHandler func(*healthpb.HealthCheckRequest, healthpb.Health_WatchServer) error

// ExecHandler
type ExecHandler func(context.Context, *executorpb.ExecuteFunctionRequest) (*executorpb.ExecuteFunctionResponse, error)

type Option func(*GrpcServer)

func New(c Config, opts ...Option) *GrpcServer {
	c.setDefaults()
	s := &GrpcServer{
		config: c,
		sem:    semaphore.NewWeighted(c.MaxRPC),
		//cm:     &sync.Mutex{},
	}

	for _, o := range opts {
		o(s)
	}

	return s
}

func (r *GrpcServer) Start(ctx context.Context) error {
	r.l = log.FromContext(ctx)
	r.l.Info("grpc server start...")
	r.l.Info("grpc server start",
		"address", r.config.Address,
		"certDir", r.config.CertDir,
		"certName", r.config.CertName,
		"keyName", r.config.KeyName,
		"caName", r.config.CaName,
	)
	l, err := net.Listen("tcp", r.config.Address)
	if err != nil {
		return errors.Wrap(err, "cannot listen")
	}
	opts, err := r.serverOpts(ctx)
	if err != nil {
		return err
	}
	// create a gRPC server object
	grpcServer := grpc.NewServer(opts...)

	reflection.Register(grpcServer)

	executorpb.RegisterFunctionExecutorServer(grpcServer, r)
	r.l.Info("grpc server with exec function...")

	healthpb.RegisterHealthServer(grpcServer, r)
	r.l.Info("grpc server with health...")

	r.l.Info("starting grpc server...")
	err = grpcServer.Serve(l)
	if err != nil {
		r.l.Info("gRPC serve failed", "error", err)
		return err
	}
	return nil
}

func WithCheckHandler(h CheckHandler) func(*GrpcServer) {
	return func(r *GrpcServer) {
		r.checkHandler = h
	}
}

func WithWatchHandler(h WatchHandler) func(*GrpcServer) {
	return func(r *GrpcServer) {
		r.watchHandler = h
	}
}

func WithExecHandler(h ExecHandler) func(*GrpcServer) {
	return func(r *GrpcServer) {
		r.execHandler = h
	}
}

func (r *GrpcServer) acquireSem(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return r.sem.Acquire(ctx, 1)
	}
}
