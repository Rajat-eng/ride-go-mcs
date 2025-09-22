package main

import "math/rand"

// Predefined routes for drivers (used for the gRPC Streaming module)
// (these are San Francisco routes, get these coordinates from Google Maps for example and build a custom route if you want)
var PredefinedRoutes = [][][]float64{
	{ // Route 1 - Around Marathahalli
		{12.9550, 77.7010},
		{12.9585, 77.7072},
		{12.9610, 77.7120},
		{12.9635, 77.7185},
	},

	{ // Route 2 - From Bellandur to Kadubeesanahalli
		{12.9300, 77.6780},
		{12.9350, 77.6855},
		{12.9400, 77.6910},
		{12.9445, 77.6965},
		{12.9500, 77.7020},
	},

	{ // Route 3 - Outer Ring Road near Ecospace
		{12.9240, 77.6785},
		{12.9265, 77.6830},
		{12.9300, 77.6875},
		{12.9330, 77.6920},
		{12.9360, 77.6965},
		{12.9385, 77.7010},
	},

	{ // Route 4 - Sarjapur Road area
		{12.9355, 77.7100},
		{12.9380, 77.7145},
		{12.9420, 77.7190},
		{12.9450, 77.7235},
		{12.9480, 77.7280},
	},
}

func GenerateRandomPlate() string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	plate := ""
	for i := 0; i < 3; i++ {
		plate += string(letters[rand.Intn(len(letters))])
	}

	return plate
}
