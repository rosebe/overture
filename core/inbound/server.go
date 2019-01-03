// Package inbound implements dns server for inbound connection.
package inbound

import (
	"net"
	"os"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/miekg/dns"
	"github.com/shawn1m/overture/core/outbound"
)

type Server struct {
	BindAddress string

	Dispatcher outbound.Dispatcher

	RejectQtype []uint16
}

func (s *Server) Run() {

	mux := dns.NewServeMux()
	mux.Handle(".", s)

	wg := new(sync.WaitGroup)
	wg.Add(2)

	log.Info("Start overture on " + s.BindAddress)

	for _, p := range [2]string{"tcp", "udp"} {
		go func(p string) {
			err := dns.ListenAndServe(s.BindAddress, p, mux)
			if err != nil {
				log.Fatal("Listen "+p+" failed: ", err)
				os.Exit(1)
			}
		}(p)
	}

	wg.Wait()
}

func (s *Server) ServeDNS(w dns.ResponseWriter, q *dns.Msg) {

	inboundIP, _, _ := net.SplitHostPort(w.RemoteAddr().String())
	s.Dispatcher.InboundIP = inboundIP
	s.Dispatcher.QuestionMessage = q

	log.Debug("Question from " + inboundIP + ": " + q.Question[0].String())

	for _, qt := range s.RejectQtype {
		if isQuestionType(q, qt) {
			return
		}
	}

	d := s.Dispatcher

	d.Exchange()

	cb := d.ActiveClientBundle

	if cb.ResponseMessage == nil {
		return
	}

	err := w.WriteMsg(cb.ResponseMessage)
	if err != nil {
		log.Warn("Write message fail:", cb.ResponseMessage)
		return
	}
}

func isQuestionType(q *dns.Msg, qt uint16) bool { return q.Question[0].Qtype == qt }
