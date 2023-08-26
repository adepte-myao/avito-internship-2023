package server

import (
	"net/http"
	"time"

	"avito-internship-2023/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

type Server struct {
	http.Server
	logger common.Logger
	router *gin.Engine
}

func NewServer(logger common.Logger, router *gin.Engine, addr string) *Server {
	serv := &Server{
		logger: logger,
		router: router,
	}

	serv.configure(addr)

	return serv
}

func (s *Server) Start() error {
	s.logger.Info("Server started")

	err := s.ListenAndServe()
	return err
}

func (s *Server) configure(addr string) {
	s.Addr = addr
	s.Handler = s.router
	s.IdleTimeout = 5 * time.Second
	s.ReadTimeout = 5 * time.Second
	s.WriteTimeout = 5 * time.Second
}
