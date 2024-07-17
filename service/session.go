package service

import (
	"a21hc3NpZ25tZW50/model"
	"a21hc3NpZ25tZW50/repository"
)

type SessionService interface {
	GetSessionByEmail(email string) (model.Session, error)
}

type sessionService struct {
	sessionRepo repository.SessionRepository
}

func NewSessionService(sessionRepo repository.SessionRepository) *sessionService {
	return &sessionService{sessionRepo: sessionRepo}
}

func (s *sessionService) GetSessionByEmail(email string) (model.Session, error) {
	session, err := s.sessionRepo.SessionAvailEmail(email)
	if err != nil {
		return model.Session{}, err
	}
	return session, nil
}