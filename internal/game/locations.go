package game

import "math/rand"

type Location struct {
	Name string
	Lat  float64
	Lng  float64
}

var Locations = []Location{
	{"Japan", 36.2048, 138.2529},
	{"Brazil", -14.2350, -51.9253},
	{"Egypt", 26.8206, 30.8025},
	{"Canada", 56.1304, -106.3468},
	{"Australia", -25.2744, 133.7751},
	{"France", 46.2276, 2.2137},
	{"India", 20.5937, 78.9629},
	{"South Africa", -30.5595, 22.9375},
	{"Mexico", 23.6345, -102.5528},
	{"Norway", 60.4720, 8.4689},
	{"Thailand", 15.8700, 100.9925},
	{"Argentina", -38.4161, -63.6167},
	{"Kenya", -0.0236, 37.9062},
	{"New Zealand", -40.9006, 174.8860},
	{"Iceland", 64.9631, -19.0208},
	{"South Korea", 35.9078, 127.7669},
	{"Peru", -9.1900, -75.0152},
	{"Morocco", 31.7917, -7.0926},
	{"Vietnam", 14.0583, 108.2772},
	{"Finland", 61.9241, 25.7482},
}

func RandomLocation() Location {
	return Locations[rand.Intn(len(Locations))]
}
