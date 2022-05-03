package web

import (
	"ensemble/storage/structures"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

type EnsembleContext struct {
	echo.Context
	session       *structures.Session
	user          *structures.User
	project       *structures.Project
	projectUpdate *structures.ProjectUpdate
	playbook      *structures.Playbook
	playbookRun   *structures.PlaybookRun
	userControl   *structures.User
}

func (c *EnsembleContext) GetSessionId() string {
	cookie, err := c.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (c *EnsembleContext) SetSessionId(id string) {
	cookie := &http.Cookie{
		Name:    SessionCookieName,
		Value:   id,
		Expires: time.Now().Add(24 * time.Hour),
	}
	c.SetCookie(cookie)
}

func (c *EnsembleContext) DeleteSessionId() {
	cookie := &http.Cookie{
		Name:    SessionCookieName,
		Value:   "",
		Expires: time.Now().Add(-24 * time.Hour),
	}
	c.SetCookie(cookie)
}
