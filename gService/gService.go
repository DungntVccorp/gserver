package gService

import (
	"gitlab.vivas.vn/go/gserver/gBase"
	"gitlab.vivas.vn/go/gserver/gHTTP"
	"gitlab.vivas.vn/go/gserver/gRPC"
	"gitlab.vivas.vn/go/internal/logger"
	"os"
	"os/signal"
)

type CallbackRequest func(*gBase.Payload)
type gService struct {
	done chan struct{}
	interrupt chan os.Signal
	receiveRequest chan *gBase.Payload
	Logger *logger.Logger
	cb CallbackRequest

	http_server *gHTTP.HTTPServer
	grpc_server *gRPC.GRPCServer
}
func New(_log *logger.Logger, configs ...gBase.ConfigOption) *gService {
	p := &gService{Logger: _log}
	p.done = make(chan struct{})
	p.receiveRequest = make(chan *gBase.Payload,100)
	p.interrupt = make(chan os.Signal, 1)
	signal.Notify(p.interrupt, os.Interrupt)


	for _,cf := range configs{
		if cf.Protocol == gBase.RequestProtocol_HTTP { // init http server listen
			p.http_server = gHTTP.New(cf,p.receiveRequest)
		}else if cf.Protocol == gBase.RequestProtocol_GRPC {
			p.grpc_server = gRPC.New(cf,p.receiveRequest)
		}
	}

	return p
}
func (p *gService)Wait(){
	<-p.done
}

func (p *gService)StartListenAndReceiveRequest() chan struct{}{

	if p.http_server != nil {
		p.http_server.Serve()
	}

	if p.grpc_server != nil {
		p.grpc_server.Serve()
	}


	go func() {
	loop:
		for{
			select {
			case rq := <- p.receiveRequest:
				if p.cb != nil{
					p.cb(rq)
				}else{
					rq.ChResult <- &gBase.Result{Status: 1010} // request lạ
				}
			case <-p.done:
				break loop
			case <-p.interrupt:
				p.LogInfo("shutting down gracefully")
				break loop
			}
		}


		p.LogInfo("End Service")

		if p.http_server != nil{
			p.http_server.Close()
		}
		if p.grpc_server != nil {
			p.grpc_server.Close()
		}
		p.done <- struct{}{}


	}()


	return p.done
}

func (p *gService)registerHandler(request CallbackRequest){
	p.cb = request
}


func (p *gService) LogInfo(format string, args ...interface{}) {
	p.Logger.Log(logger.Info, "[Service] " +format, args...)
}
func (p *gService) LogDebug(format string, args ...interface{}) {
	p.Logger.Log(logger.Debug, "[Service] "+format, args...)
}
func (p *gService) LogError(format string, args ...interface{}) {
	p.Logger.Log(logger.Error, "[Service] "+format, args...)
}
