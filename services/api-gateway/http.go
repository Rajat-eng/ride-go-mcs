package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"ride-sharing/shared/types"
	"ride-sharing/shared/util"

	"github.com/go-playground/validator/v10"
)

func HandleTripPreview(w http.ResponseWriter, r *http.Request) {
	var reqBody PreviewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// validation
	if err := types.Validate.Struct(reqBody); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		errors := make([]string, len(validationErrors))
		for i, err := range validationErrors {
			errors[i] = util.FormatValidationError(err)
		}
		util.RespondWithError(w, http.StatusBadRequest, "Validation failed", errors)
		return
	}
	jsonBody, _ := json.Marshal(reqBody) // convert struct to json
	reader := bytes.NewReader(jsonBody)  // convert json to io.Reader. io.Reader is an interface which repersents a stream of data
	response, err := http.Post("http://trip-service:8083/preview", "application/json", reader)
	// response is a kind of stream - we need to read it and then close it
	if err != nil {
		fmt.Println("error in calling trip service", err)
		http.Error(w, "Failed to get trip preview", http.StatusInternalServerError)
		return
	}
	// body, err := io.ReadAll(response.Body) // reading response body as bytes

	// if err != nil {
	// 	fmt.Printf("Error reading response body: %v\n", err)
	// 	util.RespondWithError(w, http.StatusInternalServerError, "Internal server error", nil)
	// 	return
	// }

	var tripResponse map[string]interface{}
	/*
		unmarshaled into map[string]interface{}. Unmarshal receives body as []byte
		but the problem is  that it has to hold []byte in memory before unmarshaling
		so if the response body is too large, it can lead to high memory usage
		and potential out-of-memory errors

		if err := json.Unmarshal(body, &tripResponse); err != nil {
			// & is memory address operator. It passes the reference of tripResponse to unmarshal function
			// so that unmarshal can modify the original tripResponse variable
			fmt.Printf("Error parsing response JSON: %v\n", err)
			util.RespondWithError(w, http.StatusInternalServerError, "Internal server error", nil)
			return
		}*/

	if err := json.NewDecoder(response.Body).Decode(&tripResponse); err != nil {
		util.RespondWithError(w, http.StatusInternalServerError, "Invalid JSON from trip service", nil)
		return
	}

	// encoder converts go/structs directly to json and writes to response body
	// it does not hold entire json in memory
	// it is more memory efficient
	// it is used when we want to stream large json response
	// instead of reading entire json in memory and then writing to response body
	// we can directly write to response body as we encode

	//data := MyStruct{Name: "rajat"}
	//json.NewEncoder(w as io.Writer).Encode(data) --> writes JSON directly to ResponseWriter

	defer response.Body.Close()

	util.RespondWithSuccess(w, http.StatusOK, "Trip Preview", tripResponse)
}
