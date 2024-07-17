package repository

import (
	"a21hc3NpZ25tZW50/db/filebased"
	"a21hc3NpZ25tZW50/model"
)

type SessionRepository interface {
	AddSessions(session model.Session) error
	DeleteSession(token string) error
	UpdateSessions(session model.Session) error
	SessionAvailEmail(email string) (model.Session, error)
	SessionAvailToken(token string) (model.Session, error)
	TokenValidity(token string) (model.Session, error)
}

type sessionsRepo struct {
	filebasedDb *filebased.Data
}

func NewSessionsRepo(filebasedDb *filebased.Data) *sessionsRepo {
	return &sessionsRepo{filebasedDb}
}

func (u *sessionsRepo) AddSessions(session model.Session) error {
	err := u.filebasedDb.AddSession(session)
	if err != nil {
		return err
	}
	return nil
}

func (u *sessionsRepo) DeleteSession(token string) error {
	err := u.filebasedDb.DeleteSession(token)
	if err != nil {
		return err
	}
	return nil
}

func (u *sessionsRepo) UpdateSessions(session model.Session) error {
	err := u.filebasedDb.UpdateSession(session)
	if err != nil {
		return err
	}
	return nil
}

func (u *sessionsRepo) SessionAvailEmail(email string) (model.Session, error) {
	session, err := u.filebasedDb.SessionAvailEmail(email)
	if err != nil {
		return model.Session{}, err
	}
	return session, nil
}

func (u *sessionsRepo) SessionAvailToken(token string) (model.Session, error) {
	session, err := u.filebasedDb.SessionAvailToken(token)
	if err != nil {
		return model.Session{}, err
	}
	return session, nil
}

func (u *sessionsRepo) TokenValidity(token string) (model.Session, error) {
	session, err := u.filebasedDb.TokenValidity(token)
	if err != nil {
		return model.Session{}, err
	}

	if u.filebasedDb.TokenExpired(session) {
		err := u.DeleteSession(token)
		if err != nil {
			return model.Session{}, err
		}
		return model.Session{}, err
	}

	return session, nil
}