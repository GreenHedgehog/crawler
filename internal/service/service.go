package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"crawler/internal/ctxutils"
	"crawler/internal/fetcher"
	"crawler/internal/logger"
	"crawler/internal/storage"
	"crawler/rpc/crawler"

	"github.com/google/uuid"
	"github.com/twitchtv/twirp"
)

var log = logger.G().Named("server").Sugar()

var _ crawler.Crawler = (*Server)(nil)

type Server struct {
	fetcher fetcher.Interface
	storage storage.Interface
}

func NewServer(fetcher fetcher.Interface, storage storage.Interface) (*Server, error) {
	switch {
	case fetcher == nil:
		return nil, errors.New("undefined fetcher")
	case storage == nil:
		return nil, errors.New("undefined storage")
	}
	return &Server{fetcher: fetcher, storage: storage}, nil
}

func (s *Server) Fetch(ctx context.Context, in *crawler.FetchRequest) (response *crawler.FetchResponse, err error) {
	log.Infow("fetch request", "request-id", ctxutils.GetRequestID(ctx), "in", in)
	defer log.Infow("list response", "request-id", ctxutils.GetRequestID(ctx), "out", response)

	_, err = url.Parse(in.GetUrl())
	if err != nil {
		log.Errorw("url parse", "request-id", ctxutils.GetRequestID(ctx), "error", err.Error())
		return nil, twirp.NewError(twirp.InvalidArgument, "invalid url value")
	}

	products, err := s.fetcher.Fetch(ctx, in.GetUrl())
	if err != nil {
		log.Errorw("fetch", "request-id", ctxutils.GetRequestID(ctx), "error", err.Error())
		return nil, twirp.NewError(twirp.Internal, "fetch failed")
	}

	if len(products) == 0 {
		log.Infow("zero update", "request-id", ctxutils.GetRequestID(ctx))
		return &crawler.FetchResponse{}, nil
	}

	err = s.storage.Insert(ctx, products)
	if err != nil {
		log.Errorw("insert", "request-id", ctxutils.GetRequestID(ctx), "error", err.Error())
		return nil, twirp.NewError(twirp.Internal, "insert failed")
	}

	return &crawler.FetchResponse{}, nil
}

func (s *Server) List(ctx context.Context, in *crawler.ListRequest) (response *crawler.ListResponse, err error) {
	log.Infow("list request", "request-id", ctxutils.GetRequestID(ctx), "in", in)
	defer log.Infow("list response", "request-id", ctxutils.GetRequestID(ctx), "out", response)

	if in.PageSize == 0 {
		log.Errorw("zero page size requested", "request-id", ctxutils.GetRequestID(ctx))
		return nil, twirp.NewError(twirp.InvalidArgument, "invalid page size")
	}

	products, token, err := s.storage.List(ctx, in.GetSearchParams(), in.GetPageToken(), in.GetPageSize())
	if err != nil {
		log.Errorw("list", "request-id", ctxutils.GetRequestID(ctx), "error", err.Error())
		return nil, twirp.NewError(twirp.Internal, "list failed")
	}

	return &crawler.ListResponse{
		List:          products,
		NextPageToken: token,
	}, nil
}

func (s *Server) Start(addr string) error {
	log.Infow("server start", "addr", addr)

	return http.ListenAndServe(addr, crawler.NewCrawlerServer(s, twirp.WithServerHooks(&twirp.ServerHooks{
		RequestReceived: func(ctx context.Context) (context.Context, error) {
			return ctxutils.WithRequestID(ctx, uuid.NewString()), nil
		},
		Error: func(ctx context.Context, e twirp.Error) context.Context {
			log.Errorw("request failed",
				"request-id", ctxutils.GetRequestID(ctx),
				"error-code", e.Code(),
				"error-msg", e.Msg(),
			)
			return nil
		},
	})))
}
