package server

import (
	"fmt"
	"net/http"
	"net/mail"

	"github.com/eldelto/core/internal/mealplanner"
	"github.com/eldelto/core/internal/web"
	"github.com/google/uuid"
)

var (
	shareInviteTemplate        = templater.GetP("share-invite.html")
	shareInviteSuccessTemplate = templater.GetP("share-invite-success.html")
	shareInviteAcceptedTemplate = templater.GetP("share-invite-accepted.html")
)

func NewShareController(service *mealplanner.Service) *web.Controller2 {
	c := web.NewController()
	c.AddMiddleware(web.ContentTypeMiddleware(web.ContentTypeHTML))
	c.ErrorHandler = errorHandler

	c.GET("/invite", web.RenderTemplate(shareInviteTemplate, nil))
	c.POST("/invite", createShareInvite(service))
	c.GET("/invite/success", web.RenderTemplate(shareInviteSuccessTemplate, nil))
	c.GET("/invite/accept", acceptShareInvite(service))
	c.GET("/invite/accepted", web.RenderTemplate(shareInviteAcceptedTemplate, nil))

	return c
}

func createShareInvite(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return err
		}

		email, err := mail.ParseAddress(r.PostForm.Get("email"))
		if err != nil {
			return fmt.Errorf("create share invite: %w", err)
		}

		if err := service.InviteUserToShare(r.Context(), *email); err != nil {
			return err
		}

		http.Redirect(w, r, "./invite/success", http.StatusSeeOther)
		return nil
	}
}

func acceptShareInvite(service *mealplanner.Service) web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		otherUserID, err := uuid.Parse(r.URL.Query().Get("user"))
		if err != nil {
			return fmt.Errorf("accept share invite: %w", err)
		}

		if err := service.AcceptShareInvite(r.Context(), web.UserID{UUID: otherUserID}); err != nil {
			return err
		}

		http.Redirect(w, r, "./invite/accepted", http.StatusSeeOther)
		return nil
	}
}
