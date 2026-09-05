package service

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/drobyshevv/url-shortening-service/internal/model"
)

type UrlRepository interface {
	Create(context.Context, string, string) (*model.Url, error)
	Get(context.Context, string) (*model.Url, error)
	Update(context.Context, string, string) (*model.Url, error)
	Delete(context.Context, string) error
	GetStatistics(context.Context, string) (*model.UrlStats, error)
	ExistsByURL(context.Context, string) (bool, error)
}

type ServiceUrl struct {
	repository UrlRepository
}

func NewServiceUrl(repository UrlRepository) *ServiceUrl {
	return &ServiceUrl{
		repository: repository,
	}
}

func (s *ServiceUrl) PostUrl(ctx context.Context, url string) (*model.Url, error) {
	code := randStringBytesRmndr()

	ex, err := s.repository.ExistsByURL(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("url create failed: %w", err)
	}

	if ex {
		return nil, model.ErrAlreadyExists
	}

	resp, err := s.repository.Create(ctx, url, code)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// TODO AccessCount++
func (s *ServiceUrl) GetUrl(ctx context.Context, code string) (*model.Url, error) {
	resp, err := s.repository.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *ServiceUrl) PutUrl(ctx context.Context, code string, url string) (*model.Url, error) {
	resp, err := s.repository.Update(ctx, code, url)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *ServiceUrl) DeleteUrl(ctx context.Context, code string) error {
	err := s.repository.Delete(ctx, code)
	if err != nil {
		return err
	}
	return nil
}

func (s *ServiceUrl) GetUrlStatistics(ctx context.Context, code string) (*model.UrlStats, error) {
	resp, err := s.repository.GetStatistics(ctx, code)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytesRmndr() string {
	b := make([]byte, 8)

	for i := range b {
		if i < 4 {
			b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
		} else {
			b[i] = byte('0' + rand.Intn(10))
		}
	}

	return string(b)
}
