package main

import (
	"encoding/json"
	"log"
	"net/http"
	pb "ride-sharing/shared/proto/login"
	"ride-sharing/shared/util"
)

type SignupRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=8"`
	Name        string `json:"name" validate:"required,min=1"`
	PhoneNumber string `json:"phoneNumber"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=1"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required,min=1"`
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
	})
	if err != nil {
		log.Printf("Signup error: %v", err)
		util.RespondWithError(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	util.RespondWithSuccess(w, http.StatusCreated, "Account created", resp)
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

	util.RespondWithSuccess(w, http.StatusOK, "Login successful", resp)
}

func HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	_, span := tracer.Start(r.Context(), "handleRefreshToken")
	defer span.End()

	var reqBody RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	resp, err := loginClient.Client.RefreshToken(r.Context(), &pb.RefreshTokenRequest{
		RefreshToken: reqBody.RefreshToken,
	})
	if err != nil {
		log.Printf("Refresh token error: %v", err)
		util.RespondWithError(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	util.RespondWithSuccess(w, http.StatusOK, "Token refreshed", resp)
}
