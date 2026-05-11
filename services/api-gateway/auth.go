package main

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/shared/env"
	pb "ride-sharing/shared/proto/login"
	"ride-sharing/shared/util"
	"time"
)

const refreshTokenCookie = "refresh_token"

// authResponse is what we send to the client.
// The refresh token goes into an HttpOnly cookie — never in the body.
type authResponse struct {
	AccessToken string   `json:"accessToken"`
	User        *pb.User `json:"user,omitempty"`
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	secure := env.GetString("ENVIRONMENT", "development") != "development"
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	log.Printf("refresh_token cookie set (len=%d)", len(token))
}

func clearRefreshCookie(w http.ResponseWriter) {
	secure := env.GetString("ENVIRONMENT", "development") != "development"
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

type SignupRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	Name        string `json:"name" validate:"required,min=1"`
	PhoneNumber string `json:"phoneNumber"`
	Role        string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=1"`
}

type GoogleAuthRequest struct {
	IDToken string `json:"idToken" validate:"required"`
	Role    string `json:"role"`
}

func HandleSignup(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "handleSignup")
	defer span.End()

	var reqBody SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	resp, err := loginClient.Client.Signup(r.Context(), &pb.SignupRequest{
		Email:       reqBody.Email,
		Password:    reqBody.Password,
		Name:        reqBody.Name,
		PhoneNumber: reqBody.PhoneNumber,
		Role:        reqBody.Role,
	})
	if err != nil {
		log.Printf("Signup error: %v", err)
		util.RespondWithError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	setRefreshCookie(w, resp.RefreshToken)
	util.RespondWithSuccess(w, http.StatusCreated, "Account created", authResponse{
		AccessToken: resp.AccessToken,
		User:        resp.User,
	})
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "handleLogin")
	defer span.End()

	var reqBody LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	resp, err := loginClient.Client.Login(r.Context(), &pb.LoginRequest{
		Email:    reqBody.Email,
		Password: reqBody.Password,
	})
	if err != nil {
		log.Printf("Login error: %v", err)
		util.RespondWithError(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	setRefreshCookie(w, resp.RefreshToken)
	util.RespondWithSuccess(w, http.StatusOK, "Login successful", authResponse{
		AccessToken: resp.AccessToken,
		User:        resp.User,
	})
}

// HandleRefreshToken reads the refresh token from the HttpOnly cookie set at login.
// No request body is needed — the browser sends the cookie automatically.
func HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "handleRefreshToken")
	defer span.End()

	cookie, err := r.Cookie(refreshTokenCookie)
	if err != nil || cookie.Value == "" {
		http.Error(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	resp, err := loginClient.Client.RefreshToken(r.Context(), &pb.RefreshTokenRequest{
		RefreshToken: cookie.Value,
	})
	if err != nil {
		log.Printf("Refresh token error: %v", err)
		clearRefreshCookie(w)
		util.RespondWithError(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	setRefreshCookie(w, resp.RefreshToken)
	util.RespondWithSuccess(w, http.StatusOK, "Token refreshed", authResponse{
		AccessToken: resp.AccessToken,
		User:        resp.User,
	})
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	clearRefreshCookie(w)
	util.RespondWithSuccess(w, http.StatusOK, "Logged out", nil)
}

func HandleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "handleGoogleAuth")
	defer span.End()

	var reqBody GoogleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	resp, err := loginClient.Client.GoogleAuth(r.Context(), &pb.GoogleAuthRequest{
		IdToken: reqBody.IDToken,
		Role:    reqBody.Role,
	})
	if err != nil {
		log.Printf("Google auth error: %v", err)
		util.RespondWithError(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}
	setRefreshCookie(w, resp.RefreshToken)
	util.RespondWithSuccess(w, http.StatusOK, "Google login successful", authResponse{
		AccessToken: resp.AccessToken,
		User:        resp.User,
	})
}
